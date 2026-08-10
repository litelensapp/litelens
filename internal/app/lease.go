package app

import (
	"context"
	"fmt"
	"log"
	"strings"

	"github.com/gknguyen/litelens/internal/dto"
	"github.com/gknguyen/litelens/internal/kube/resources"
	"github.com/wailsapp/wails/v2/pkg/runtime"
	coordinationv1 "k8s.io/api/coordination/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	sigsyaml "sigs.k8s.io/yaml"
)

func (a *App) ListLeases(namespace string) ([]dto.Lease, error) {
	a.mu.RLock()
	h := a.factories[a.activeContext]
	a.mu.RUnlock()
	if h == nil {
		return []dto.Lease{}, nil
	}
	if h.IsForbidden("leases") {
		return []dto.Lease{}, nil
	}
	<-h.GetSyncedChan("leases")
	if h.IsForbidden("leases") {
		return nil, nil
	}
	result, err := kubeResources.ListLeases(h.Factory.Coordination().V1().Leases().Lister(), namespace)
	if err != nil {
		log.Printf("app: ListLeases: %v", err)
		return []dto.Lease{}, nil
	}
	return result, nil
}

func (a *App) GetLeaseByName(namespace, name string) (dto.Lease, error) {
	a.mu.RLock()
	h := a.factories[a.activeContext]
	a.mu.RUnlock()
	if h == nil {
		return dto.Lease{}, nil
	}
	if h.IsForbidden("leases") {
		return dto.Lease{}, nil
	}
	<-h.GetSyncedChan("leases")
	if h.IsForbidden("leases") {
		return dto.Lease{}, nil
	}
	result, err := kubeResources.GetLeaseByName(h.Factory.Coordination().V1().Leases().Lister(), namespace, name)
	if err != nil {
		log.Printf("app: GetLeaseByName: %v", err)
		return dto.Lease{}, nil
	}
	return result, nil
}

func (a *App) DeleteLease(namespace, name string) error {
	a.mu.RLock()
	cs := a.clients[a.activeContext]
	a.mu.RUnlock()
	if cs == nil {
		return fmt.Errorf("not connected")
	}

	ctx, cancel := context.WithTimeout(context.Background(), apiMutationTimeout)
	defer cancel()
	err := cs.CoordinationV1().Leases(namespace).Delete(ctx, name, metav1.DeleteOptions{})
	if err != nil && !errors.IsNotFound(err) {
		return fmt.Errorf("delete Lease: %w", err)
	}

	a.emitLeases(namespace)

	return nil
}

func (a *App) DeleteLeases(items []dto.LeaseRef) error {
	a.mu.RLock()
	cs := a.clients[a.activeContext]
	a.mu.RUnlock()
	if cs == nil {
		return fmt.Errorf("not connected")
	}

	var msgs []string
	namespaces := make(map[string]bool)

	for _, ref := range items {
		ctx, cancel := context.WithTimeout(context.Background(), apiMutationTimeout)
		err := cs.CoordinationV1().Leases(ref.Namespace).Delete(ctx, ref.Name, metav1.DeleteOptions{})
		cancel()
		if err != nil && !errors.IsNotFound(err) {
			msgs = append(msgs, fmt.Sprintf("%s/%s: %v", ref.Namespace, ref.Name, err))
		}
		namespaces[ref.Namespace] = true
	}

	// Emit updates for each unique namespace touched
	for ns := range namespaces {
		a.emitLeases(ns)
	}

	if len(msgs) > 0 {
		return fmt.Errorf("failed to delete %d of %d leases: %s", len(msgs), len(items), strings.Join(msgs, "; "))
	}
	return nil
}

func (a *App) emitLeases(namespace string) {
	a.mu.RLock()
	h := a.factories[a.activeContext]
	a.mu.RUnlock()
	if h == nil {
		return
	}
	if h.IsForbidden("leases") {
		return
	}
	<-h.GetSyncedChan("leases")
	if h.IsForbidden("leases") {
		return
	}
	lister := h.Factory.Coordination().V1().Leases().Lister()
	allData, err := kubeResources.ListLeases(lister, "")
	if err != nil {
		log.Printf("app: emitLeases: %v", err)
		return
	}
	runtime.EventsEmit(a.ctx, "leases:update", allData)
	if namespace != "" {
		nsData, err := kubeResources.ListLeases(lister, namespace)
		if err != nil {
			log.Printf("app: emitLeases ns=%s: %v", namespace, err)
			return
		}
		runtime.EventsEmit(a.ctx, "leases:"+namespace+":update", nsData)
	}
}

func (a *App) GetLeaseYAML(namespace, name string) (string, error) {
	a.mu.RLock()
	cs := a.clients[a.activeContext]
	a.mu.RUnlock()
	if cs == nil {
		return "", fmt.Errorf("not connected")
	}

	ctx, cancel := context.WithTimeout(context.Background(), apiReadTimeout)
	defer cancel()
	lease, err := cs.CoordinationV1().Leases(namespace).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		return "", fmt.Errorf("get Lease: %w", err)
	}

	b, err := sigsyaml.Marshal(lease)
	if err != nil {
		return "", fmt.Errorf("marshal Lease: %w", err)
	}

	return string(b), nil
}

func (a *App) UpdateLeaseYAML(namespace, yamlString string) error {
	a.mu.RLock()
	cs := a.clients[a.activeContext]
	a.mu.RUnlock()
	if cs == nil {
		return fmt.Errorf("not connected")
	}

	var lease coordinationv1.Lease
	err := sigsyaml.Unmarshal([]byte(yamlString), &lease)
	if err != nil {
		return fmt.Errorf("unmarshal YAML to Lease: %w", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), apiMutationTimeout)
	defer cancel()
	_, err = cs.CoordinationV1().Leases(namespace).Update(ctx, &lease, metav1.UpdateOptions{})
	if err != nil {
		return fmt.Errorf("update Lease: %w", err)
	}

	a.emitLeases(namespace)

	return nil
}

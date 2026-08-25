package app

import (
	"context"
	"fmt"
	"log"
	"strings"

	kubeResources "github.com/litelensapp/litelens/internal/kube/resources"
	"github.com/litelensapp/litelens/packages/core/dto"
	"github.com/wailsapp/wails/v2/pkg/runtime"
	coordinationv1 "k8s.io/api/coordination/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	sigsyaml "sigs.k8s.io/yaml"
)

func (a *App) ListLeases() ([]dto.Lease, error) {
	a.mu.RLock()
	h := a.factories[a.activeContext]
	namespaces := a.activeNamespaces
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
	result, err := kubeResources.ListLeases(h.Factory.Coordination().V1().Leases().Lister(), namespaces)
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

	a.emitLeases()

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

	for _, ref := range items {
		ctx, cancel := context.WithTimeout(context.Background(), apiMutationTimeout)
		err := cs.CoordinationV1().Leases(ref.Namespace).Delete(ctx, ref.Name, metav1.DeleteOptions{})
		cancel()
		if err != nil && !errors.IsNotFound(err) {
			msgs = append(msgs, fmt.Sprintf("%s/%s: %v", ref.Namespace, ref.Name, err))
		}
	}

	a.emitLeases()

	if len(msgs) > 0 {
		return fmt.Errorf("failed to delete %d of %d leases: %s", len(msgs), len(items), strings.Join(msgs, "; "))
	}
	return nil
}

func (a *App) emitLeases() {
	a.mu.RLock()
	h := a.factories[a.activeContext]
	namespaces := a.activeNamespaces
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
	data, err := kubeResources.ListLeases(lister, namespaces)
	if err != nil {
		log.Printf("app: emitLeases: %v", err)
		return
	}
	runtime.EventsEmit(a.ctx, "leases:update", data)
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

	a.emitLeases()

	return nil
}

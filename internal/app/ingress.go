package app

import (
	"context"
	"fmt"
	"log"
	"strings"

	"github.com/litelensapp/litelens/packages/core/dto"
	"github.com/litelensapp/litelens/internal/kube/resources"
	"github.com/wailsapp/wails/v2/pkg/runtime"
	networkingv1 "k8s.io/api/networking/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	sigsyaml "sigs.k8s.io/yaml"
)

func (a *App) ListIngresses(namespace string) ([]dto.Ingress, error) {
	a.mu.RLock()
	h := a.factories[a.activeContext]
	a.mu.RUnlock()
	if h == nil {
		return []dto.Ingress{}, nil
	}
	if h.IsForbidden("ingresses") {
		return []dto.Ingress{}, nil
	}
	<-h.GetSyncedChan("ingresses")
	if h.IsForbidden("ingresses") {
		return nil, nil
	}
	result, err := kubeResources.ListIngresses(h.Factory.Networking().V1().Ingresses().Lister(), namespace)
	if err != nil {
		log.Printf("app: ListIngresses: %v", err)
		return []dto.Ingress{}, nil
	}
	return result, nil
}

func (a *App) GetIngressByName(namespace, name string) (dto.IngressDetail, error) {
	a.mu.RLock()
	h := a.factories[a.activeContext]
	a.mu.RUnlock()
	if h == nil {
		return dto.IngressDetail{}, nil
	}
	if h.IsForbidden("ingresses") {
		return dto.IngressDetail{}, nil
	}
	<-h.GetSyncedChan("ingresses")
	if h.IsForbidden("ingresses") {
		return dto.IngressDetail{}, nil
	}
	result, err := kubeResources.GetIngressByName(h.Factory.Networking().V1().Ingresses().Lister(), namespace, name)
	if err != nil {
		log.Printf("app: GetIngressByName: %v", err)
		return dto.IngressDetail{}, nil
	}
	return result, nil
}

// DeleteIngress deletes an Ingress from the specified namespace.
func (a *App) DeleteIngress(namespace, name string) error {
	a.mu.RLock()
	cs := a.clients[a.activeContext]
	a.mu.RUnlock()
	if cs == nil {
		return fmt.Errorf("not connected")
	}

	ctx, cancel := context.WithTimeout(context.Background(), apiMutationTimeout)
	defer cancel()
	err := cs.NetworkingV1().Ingresses(namespace).Delete(ctx, name, metav1.DeleteOptions{})
	if err != nil && !errors.IsNotFound(err) {
		return fmt.Errorf("delete Ingress: %w", err)
	}

	// Emit update event after successful delete
	a.emitIngresses(namespace)

	return nil
}

// DeleteIngresses deletes multiple Ingresses, handling best-effort deletion across namespaces.
func (a *App) DeleteIngresses(items []dto.IngressRef) error {
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
		err := cs.NetworkingV1().Ingresses(ref.Namespace).Delete(ctx, ref.Name, metav1.DeleteOptions{})
		cancel()
		if err != nil && !errors.IsNotFound(err) {
			msgs = append(msgs, fmt.Sprintf("%s/%s: %v", ref.Namespace, ref.Name, err))
		}
		namespaces[ref.Namespace] = true
	}

	// Emit updates for each unique namespace touched
	for ns := range namespaces {
		a.emitIngresses(ns)
	}

	if len(msgs) > 0 {
		return fmt.Errorf("failed to delete %d of %d ingresses: %s", len(msgs), len(items), strings.Join(msgs, "; "))
	}
	return nil
}

func (a *App) emitIngresses(namespace string) {
	a.mu.RLock()
	h := a.factories[a.activeContext]
	a.mu.RUnlock()
	if h == nil {
		return
	}
	if h.IsForbidden("ingresses") {
		return
	}
	<-h.GetSyncedChan("ingresses")
	if h.IsForbidden("ingresses") {
		return
	}
	lister := h.Factory.Networking().V1().Ingresses().Lister()
	allData, err := kubeResources.ListIngresses(lister, "")
	if err != nil {
		log.Printf("app: emitIngresses: %v", err)
		return
	}
	runtime.EventsEmit(a.ctx, "ingresses:update", allData)
	if namespace != "" {
		// Filter already-fetched cluster-wide data instead of re-listing
		nsData := make([]dto.Ingress, 0)
		for _, item := range allData {
			if item.Namespace == namespace {
				nsData = append(nsData, item)
			}
		}
		runtime.EventsEmit(a.ctx, "ingresses:"+namespace+":update", nsData)
	}
}

func (a *App) GetIngressYAML(namespace, name string) (string, error) {
	a.mu.RLock()
	cs := a.clients[a.activeContext]
	a.mu.RUnlock()
	if cs == nil {
		return "", fmt.Errorf("not connected")
	}

	ctx, cancel := context.WithTimeout(context.Background(), apiReadTimeout)
	defer cancel()
	ing, err := cs.NetworkingV1().Ingresses(namespace).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		return "", fmt.Errorf("get Ingress: %w", err)
	}

	b, err := sigsyaml.Marshal(ing)
	if err != nil {
		return "", fmt.Errorf("marshal Ingress to YAML: %w", err)
	}

	return string(b), nil
}

func (a *App) UpdateIngressYAML(namespace, yamlString string) error {
	a.mu.RLock()
	cs := a.clients[a.activeContext]
	a.mu.RUnlock()
	if cs == nil {
		return fmt.Errorf("not connected")
	}

	var ing networkingv1.Ingress
	err := sigsyaml.Unmarshal([]byte(yamlString), &ing)
	if err != nil {
		return fmt.Errorf("unmarshal YAML to Ingress: %w", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), apiMutationTimeout)
	defer cancel()
	_, err = cs.NetworkingV1().Ingresses(namespace).Update(ctx, &ing, metav1.UpdateOptions{})
	if err != nil {
		return fmt.Errorf("update Ingress: %w", err)
	}

	a.emitIngresses(namespace)

	return nil
}

package app

import (
	"context"
	"fmt"
	"log"
	"strings"

	"github.com/litelensapp/litelens/packages/core/dto"
	"github.com/litelensapp/litelens/internal/kube/resources"
	"github.com/wailsapp/wails/v2/pkg/runtime"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	sigsyaml "sigs.k8s.io/yaml"
)

func (a *App) ListEndpoints(namespace string) ([]dto.Endpoint, error) {
	a.mu.RLock()
	h := a.factories[a.activeContext]
	a.mu.RUnlock()
	if h == nil {
		return []dto.Endpoint{}, nil
	}
	if h.IsForbidden("endpoints") {
		return []dto.Endpoint{}, nil
	}
	<-h.GetSyncedChan("endpoints")
	if h.IsForbidden("endpoints") {
		return nil, nil
	}
	result, err := kubeResources.ListEndpoints(h.Factory.Core().V1().Endpoints().Lister(), namespace)
	if err != nil {
		log.Printf("app: ListEndpoints: %v", err)
		return []dto.Endpoint{}, nil
	}
	return result, nil
}

func (a *App) GetEndpointByName(namespace, name string) (dto.Endpoint, error) {
	a.mu.RLock()
	h := a.factories[a.activeContext]
	a.mu.RUnlock()
	if h == nil {
		return dto.Endpoint{}, nil
	}
	if h.IsForbidden("endpoints") {
		return dto.Endpoint{}, nil
	}
	<-h.GetSyncedChan("endpoints")
	if h.IsForbidden("endpoints") {
		return dto.Endpoint{}, nil
	}
	result, err := kubeResources.GetEndpointByName(h.Factory.Core().V1().Endpoints().Lister(), namespace, name)
	if err != nil {
		log.Printf("app: GetEndpointByName: %v", err)
		return dto.Endpoint{}, nil
	}
	return result, nil
}

// DeleteEndpoint deletes an Endpoint from the specified namespace.
func (a *App) DeleteEndpoint(namespace, name string) error {
	a.mu.RLock()
	cs := a.clients[a.activeContext]
	a.mu.RUnlock()
	if cs == nil {
		return fmt.Errorf("not connected")
	}

	ctx, cancel := context.WithTimeout(context.Background(), apiMutationTimeout)
	defer cancel()
	err := cs.CoreV1().Endpoints(namespace).Delete(ctx, name, metav1.DeleteOptions{})
	if err != nil && !errors.IsNotFound(err) {
		return fmt.Errorf("delete Endpoint: %w", err)
	}

	// Emit update event after successful delete
	a.emitEndpoints(namespace)

	return nil
}

// DeleteEndpoints deletes multiple Endpoints, handling best-effort deletion across namespaces.
func (a *App) DeleteEndpoints(items []dto.EndpointRef) error {
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
		err := cs.CoreV1().Endpoints(ref.Namespace).Delete(ctx, ref.Name, metav1.DeleteOptions{})
		cancel()
		if err != nil && !errors.IsNotFound(err) {
			msgs = append(msgs, fmt.Sprintf("%s/%s: %v", ref.Namespace, ref.Name, err))
		}
		namespaces[ref.Namespace] = true
	}

	// Emit updates for each unique namespace touched
	for ns := range namespaces {
		a.emitEndpoints(ns)
	}

	if len(msgs) > 0 {
		return fmt.Errorf("failed to delete %d of %d endpoints: %s", len(msgs), len(items), strings.Join(msgs, "; "))
	}
	return nil
}

func (a *App) emitEndpoints(namespace string) {
	a.mu.RLock()
	h := a.factories[a.activeContext]
	a.mu.RUnlock()
	if h == nil {
		return
	}
	if h.IsForbidden("endpoints") {
		return
	}
	<-h.GetSyncedChan("endpoints")
	if h.IsForbidden("endpoints") {
		return
	}
	lister := h.Factory.Core().V1().Endpoints().Lister()
	allData, err := kubeResources.ListEndpoints(lister, "")
	if err != nil {
		log.Printf("app: emitEndpoints: %v", err)
		return
	}
	runtime.EventsEmit(a.ctx, "endpoints:update", allData)
	if namespace != "" {
		// Filter already-fetched cluster-wide data instead of re-listing
		nsData := make([]dto.Endpoint, 0)
		for _, item := range allData {
			if item.Namespace == namespace {
				nsData = append(nsData, item)
			}
		}
		runtime.EventsEmit(a.ctx, "endpoints:"+namespace+":update", nsData)
	}
}

func (a *App) GetEndpointYAML(namespace, name string) (string, error) {
	a.mu.RLock()
	cs := a.clients[a.activeContext]
	a.mu.RUnlock()
	if cs == nil {
		return "", fmt.Errorf("not connected")
	}

	ctx, cancel := context.WithTimeout(context.Background(), apiReadTimeout)
	defer cancel()
	ep, err := cs.CoreV1().Endpoints(namespace).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		return "", fmt.Errorf("get Endpoint: %w", err)
	}

	b, err := sigsyaml.Marshal(ep)
	if err != nil {
		return "", fmt.Errorf("marshal Endpoint to YAML: %w", err)
	}

	return string(b), nil
}

func (a *App) UpdateEndpointYAML(namespace, yamlString string) error {
	a.mu.RLock()
	cs := a.clients[a.activeContext]
	a.mu.RUnlock()
	if cs == nil {
		return fmt.Errorf("not connected")
	}

	var ep corev1.Endpoints
	err := sigsyaml.Unmarshal([]byte(yamlString), &ep)
	if err != nil {
		return fmt.Errorf("unmarshal YAML to Endpoint: %w", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), apiMutationTimeout)
	defer cancel()
	_, err = cs.CoreV1().Endpoints(namespace).Update(ctx, &ep, metav1.UpdateOptions{})
	if err != nil {
		return fmt.Errorf("update Endpoint: %w", err)
	}

	a.emitEndpoints(namespace)

	return nil
}

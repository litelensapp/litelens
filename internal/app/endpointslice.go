package app

import (
	"context"
	"fmt"
	"log"
	"strings"

	"github.com/litelensapp/litelens/internal/dto"
	"github.com/litelensapp/litelens/internal/kube/resources"
	"github.com/wailsapp/wails/v2/pkg/runtime"
	discoveryv1 "k8s.io/api/discovery/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	sigsyaml "sigs.k8s.io/yaml"
)

func (a *App) ListEndpointSlices(namespace string) ([]dto.EndpointSlice, error) {
	a.mu.RLock()
	h := a.factories[a.activeContext]
	a.mu.RUnlock()
	if h == nil {
		return []dto.EndpointSlice{}, nil
	}
	if h.IsForbidden("endpointslices") {
		return []dto.EndpointSlice{}, nil
	}
	<-h.GetSyncedChan("endpointslices")
	if h.IsForbidden("endpointslices") {
		return nil, nil
	}
	result, err := kubeResources.ListEndpointSlices(h.Factory.Discovery().V1().EndpointSlices().Lister(), namespace)
	if err != nil {
		log.Printf("app: ListEndpointSlices: %v", err)
		return []dto.EndpointSlice{}, nil
	}
	return result, nil
}

func (a *App) GetEndpointSliceByName(namespace, name string) (dto.EndpointSlice, error) {
	a.mu.RLock()
	h := a.factories[a.activeContext]
	a.mu.RUnlock()
	if h == nil {
		return dto.EndpointSlice{}, nil
	}
	if h.IsForbidden("endpointslices") {
		return dto.EndpointSlice{}, nil
	}
	<-h.GetSyncedChan("endpointslices")
	if h.IsForbidden("endpointslices") {
		return dto.EndpointSlice{}, nil
	}
	result, err := kubeResources.GetEndpointSliceByName(h.Factory.Discovery().V1().EndpointSlices().Lister(), namespace, name)
	if err != nil {
		log.Printf("app: GetEndpointSliceByName: %v", err)
		return dto.EndpointSlice{}, nil
	}
	return result, nil
}

// DeleteEndpointSlice deletes an EndpointSlice from the specified namespace.
func (a *App) DeleteEndpointSlice(namespace, name string) error {
	a.mu.RLock()
	cs := a.clients[a.activeContext]
	a.mu.RUnlock()
	if cs == nil {
		return fmt.Errorf("not connected")
	}

	ctx, cancel := context.WithTimeout(context.Background(), apiMutationTimeout)
	defer cancel()
	err := cs.DiscoveryV1().EndpointSlices(namespace).Delete(ctx, name, metav1.DeleteOptions{})
	if err != nil && !errors.IsNotFound(err) {
		return fmt.Errorf("delete EndpointSlice: %w", err)
	}

	// Emit update event after successful delete
	a.emitEndpointSlices(namespace)

	return nil
}

// DeleteEndpointSlices deletes multiple EndpointSlices, handling best-effort deletion across namespaces.
func (a *App) DeleteEndpointSlices(items []dto.EndpointSliceRef) error {
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
		err := cs.DiscoveryV1().EndpointSlices(ref.Namespace).Delete(ctx, ref.Name, metav1.DeleteOptions{})
		cancel()
		if err != nil && !errors.IsNotFound(err) {
			msgs = append(msgs, fmt.Sprintf("%s/%s: %v", ref.Namespace, ref.Name, err))
		}
		namespaces[ref.Namespace] = true
	}

	// Emit updates for each unique namespace touched
	for ns := range namespaces {
		a.emitEndpointSlices(ns)
	}

	if len(msgs) > 0 {
		return fmt.Errorf("failed to delete %d of %d endpointslices: %s", len(msgs), len(items), strings.Join(msgs, "; "))
	}
	return nil
}

func (a *App) emitEndpointSlices(namespace string) {
	a.mu.RLock()
	h := a.factories[a.activeContext]
	a.mu.RUnlock()
	if h == nil {
		return
	}
	if h.IsForbidden("endpointslices") {
		return
	}
	<-h.GetSyncedChan("endpointslices")
	if h.IsForbidden("endpointslices") {
		return
	}
	lister := h.Factory.Discovery().V1().EndpointSlices().Lister()
	allData, err := kubeResources.ListEndpointSlices(lister, "")
	if err != nil {
		log.Printf("app: emitEndpointSlices: %v", err)
		return
	}
	runtime.EventsEmit(a.ctx, "endpointslices:update", allData)
	if namespace != "" {
		// Filter already-fetched cluster-wide data instead of re-listing
		nsData := make([]dto.EndpointSlice, 0)
		for _, item := range allData {
			if item.Namespace == namespace {
				nsData = append(nsData, item)
			}
		}
		runtime.EventsEmit(a.ctx, "endpointslices:"+namespace+":update", nsData)
	}
}

func (a *App) GetEndpointSliceYAML(namespace, name string) (string, error) {
	a.mu.RLock()
	cs := a.clients[a.activeContext]
	a.mu.RUnlock()
	if cs == nil {
		return "", fmt.Errorf("not connected")
	}

	ctx, cancel := context.WithTimeout(context.Background(), apiReadTimeout)
	defer cancel()
	es, err := cs.DiscoveryV1().EndpointSlices(namespace).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		return "", fmt.Errorf("get EndpointSlice: %w", err)
	}

	b, err := sigsyaml.Marshal(es)
	if err != nil {
		return "", fmt.Errorf("marshal EndpointSlice to YAML: %w", err)
	}

	return string(b), nil
}

func (a *App) UpdateEndpointSliceYAML(namespace, yamlString string) error {
	a.mu.RLock()
	cs := a.clients[a.activeContext]
	a.mu.RUnlock()
	if cs == nil {
		return fmt.Errorf("not connected")
	}

	var es discoveryv1.EndpointSlice
	err := sigsyaml.Unmarshal([]byte(yamlString), &es)
	if err != nil {
		return fmt.Errorf("unmarshal YAML to EndpointSlice: %w", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), apiMutationTimeout)
	defer cancel()
	_, err = cs.DiscoveryV1().EndpointSlices(namespace).Update(ctx, &es, metav1.UpdateOptions{})
	if err != nil {
		return fmt.Errorf("update EndpointSlice: %w", err)
	}

	a.emitEndpointSlices(namespace)

	return nil
}

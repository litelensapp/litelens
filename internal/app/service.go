package app

import (
	"context"
	"fmt"
	"log"
	"strings"

	"github.com/litelensapp/litelens/internal/kube/resources"
	"github.com/litelensapp/litelens/packages/core/dto"
	"github.com/wailsapp/wails/v2/pkg/runtime"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	sigsyaml "sigs.k8s.io/yaml"
)

func (a *App) GetServiceByName(namespace, name string) (dto.Service, error) {
	a.mu.RLock()
	h := a.factories[a.activeContext]
	a.mu.RUnlock()
	if h == nil {
		return dto.Service{}, nil
	}
	result, err := kubeResources.GetServiceByName(h.Factory.Core().V1().Services().Lister(), namespace, name)
	if err != nil {
		log.Printf("app: GetServiceByName: %v", err)
		return dto.Service{}, err
	}
	return result, nil
}

func (a *App) ListServices() ([]dto.Service, error) {
	a.mu.RLock()
	h := a.factories[a.activeContext]
	namespaces := a.activeNamespaces
	a.mu.RUnlock()
	if h == nil {
		return []dto.Service{}, nil
	}
	if h.IsForbidden("services") {
		return []dto.Service{}, nil
	}
	<-h.GetSyncedChan("services")
	if h.IsForbidden("services") {
		return nil, nil
	}
	result, err := kubeResources.ListServices(h.Factory.Core().V1().Services().Lister(), namespaces)
	if err != nil {
		log.Printf("app: ListServices: %v", err)
		return []dto.Service{}, nil
	}
	return result, nil
}

// DeleteService deletes a Service from the specified namespace.
func (a *App) DeleteService(namespace, name string) error {
	a.mu.RLock()
	cs := a.clients[a.activeContext]
	a.mu.RUnlock()
	if cs == nil {
		return fmt.Errorf("not connected")
	}

	ctx, cancel := context.WithTimeout(context.Background(), apiMutationTimeout)
	defer cancel()
	err := cs.CoreV1().Services(namespace).Delete(ctx, name, metav1.DeleteOptions{})
	if err != nil && !errors.IsNotFound(err) {
		return fmt.Errorf("delete Service: %w", err)
	}

	// Emit update event after successful delete
	a.emitServices()

	return nil
}

// DeleteServices deletes multiple Services, handling best-effort deletion across namespaces.
func (a *App) DeleteServices(items []dto.ServiceRef) error {
	a.mu.RLock()
	cs := a.clients[a.activeContext]
	a.mu.RUnlock()
	if cs == nil {
		return fmt.Errorf("not connected")
	}

	var msgs []string

	for _, ref := range items {
		ctx, cancel := context.WithTimeout(context.Background(), apiMutationTimeout)
		err := cs.CoreV1().Services(ref.Namespace).Delete(ctx, ref.Name, metav1.DeleteOptions{})
		cancel()
		if err != nil && !errors.IsNotFound(err) {
			msgs = append(msgs, fmt.Sprintf("%s/%s: %v", ref.Namespace, ref.Name, err))
		}
	}

	a.emitServices()

	if len(msgs) > 0 {
		return fmt.Errorf("failed to delete %d of %d services: %s", len(msgs), len(items), strings.Join(msgs, "; "))
	}
	return nil
}

func (a *App) emitServices() {
	a.mu.RLock()
	h := a.factories[a.activeContext]
	namespaces := a.activeNamespaces
	a.mu.RUnlock()
	if h == nil {
		return
	}
	if h.IsForbidden("services") {
		return
	}
	<-h.GetSyncedChan("services")
	if h.IsForbidden("services") {
		return
	}
	lister := h.Factory.Core().V1().Services().Lister()
	data, err := kubeResources.ListServices(lister, namespaces)
	if err != nil {
		log.Printf("app: emitServices: %v", err)
		return
	}
	runtime.EventsEmit(a.ctx, "services:update", data)
}

func (a *App) GetServiceYAML(namespace, name string) (string, error) {
	a.mu.RLock()
	cs := a.clients[a.activeContext]
	a.mu.RUnlock()
	if cs == nil {
		return "", fmt.Errorf("not connected")
	}

	ctx, cancel := context.WithTimeout(context.Background(), apiReadTimeout)
	defer cancel()
	svc, err := cs.CoreV1().Services(namespace).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		return "", fmt.Errorf("get Service: %w", err)
	}

	yamlBytes, err := sigsyaml.Marshal(svc)
	if err != nil {
		return "", fmt.Errorf("marshal Service to YAML: %w", err)
	}

	return string(yamlBytes), nil
}

func (a *App) UpdateServiceYAML(namespace, yamlString string) error {
	a.mu.RLock()
	cs := a.clients[a.activeContext]
	a.mu.RUnlock()
	if cs == nil {
		return fmt.Errorf("not connected")
	}

	var svc corev1.Service
	err := sigsyaml.Unmarshal([]byte(yamlString), &svc)
	if err != nil {
		return fmt.Errorf("unmarshal YAML to Service: %w", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), apiMutationTimeout)
	defer cancel()
	_, err = cs.CoreV1().Services(namespace).Update(ctx, &svc, metav1.UpdateOptions{})
	if err != nil {
		return fmt.Errorf("update Service: %w", err)
	}

	a.emitServices()

	return nil
}

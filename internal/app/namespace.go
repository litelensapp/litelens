package app

import (
	"context"
	"fmt"
	"log"
	"strings"

	"github.com/litelensapp/litelens/internal/dto"
	"github.com/litelensapp/litelens/internal/kube/resources"
	"github.com/wailsapp/wails/v2/pkg/runtime"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	sigsyaml "sigs.k8s.io/yaml"
)

func (a *App) ListNamespaces() ([]dto.Namespace, error) {
	a.mu.RLock()
	h := a.factories[a.activeContext]
	a.mu.RUnlock()
	if h == nil {
		return []dto.Namespace{}, nil
	}
	if h.IsForbidden("namespaces") {
		return []dto.Namespace{}, nil
	}
	<-h.GetSyncedChan("namespaces")
	if h.IsForbidden("namespaces") {
		return nil, nil
	}
	result, err := kubeResources.ListNamespaces(h.Factory.Core().V1().Namespaces().Lister())
	if err != nil {
		log.Printf("app: ListNamespaces: %v", err)
		return []dto.Namespace{}, nil
	}
	return result, nil
}

func (a *App) GetNamespaceByName(name string) (dto.Namespace, error) {
	a.mu.RLock()
	h := a.factories[a.activeContext]
	a.mu.RUnlock()
	if h == nil {
		return dto.Namespace{}, nil
	}
	if h.IsForbidden("namespaces") {
		return dto.Namespace{}, nil
	}
	<-h.GetSyncedChan("namespaces")
	if h.IsForbidden("namespaces") {
		return dto.Namespace{}, nil
	}
	result, err := kubeResources.GetNamespaceByName(h.Factory.Core().V1().Namespaces().Lister(), name)
	if err != nil {
		log.Printf("app: GetNamespaceByName: %v", err)
		return dto.Namespace{}, nil
	}
	return result, nil
}

func (a *App) emitNamespaces() {
	a.mu.RLock()
	h := a.factories[a.activeContext]
	a.mu.RUnlock()
	if h == nil {
		return
	}
	if h.IsForbidden("namespaces") {
		return
	}
	<-h.GetSyncedChan("namespaces")
	if h.IsForbidden("namespaces") {
		return
	}
	data, err := kubeResources.ListNamespaces(h.Factory.Core().V1().Namespaces().Lister())
	if err != nil {
		log.Printf("app: emitNamespaces: %v", err)
		return
	}
	runtime.EventsEmit(a.ctx, "namespaces:update", data)
}

// GetNamespaces returns the list of namespace names for use in the UI selector.
func (a *App) GetNamespaces() ([]string, error) {
	nsDTOs, err := a.ListNamespaces()
	if err != nil {
		return nil, err
	}
	names := make([]string, len(nsDTOs))
	for i, ns := range nsDTOs {
		names[i] = ns.Name
	}
	return names, nil
}

func (a *App) DeleteNamespace(name string) error {
	a.mu.RLock()
	cs := a.clients[a.activeContext]
	a.mu.RUnlock()
	if cs == nil {
		return fmt.Errorf("not connected")
	}

	ctx, cancel := context.WithTimeout(context.Background(), apiMutationTimeout)
	defer cancel()
	err := cs.CoreV1().Namespaces().Delete(ctx, name, metav1.DeleteOptions{})
	if err != nil && !errors.IsNotFound(err) {
		return fmt.Errorf("delete Namespace: %w", err)
	}

	a.emitNamespaces()

	return nil
}

func (a *App) DeleteNamespaces(names []string) error {
	a.mu.RLock()
	cs := a.clients[a.activeContext]
	a.mu.RUnlock()
	if cs == nil {
		return fmt.Errorf("not connected")
	}

	var msgs []string
	for _, name := range names {
		ctx, cancel := context.WithTimeout(context.Background(), apiMutationTimeout)
		err := cs.CoreV1().Namespaces().Delete(ctx, name, metav1.DeleteOptions{})
		cancel()
		if err != nil && !errors.IsNotFound(err) {
			msgs = append(msgs, fmt.Sprintf("%s: %v", name, err))
		}
	}

	a.emitNamespaces()

	if len(msgs) > 0 {
		return fmt.Errorf("failed to delete %d of %d namespaces: %s", len(msgs), len(names), strings.Join(msgs, "; "))
	}
	return nil
}

func (a *App) CreateNamespace(name string) error {
	a.mu.RLock()
	cs := a.clients[a.activeContext]
	a.mu.RUnlock()
	if cs == nil {
		return fmt.Errorf("not connected")
	}

	ns := &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
		},
	}

	ctx, cancel := context.WithTimeout(context.Background(), apiMutationTimeout)
	defer cancel()
	_, err := cs.CoreV1().Namespaces().Create(ctx, ns, metav1.CreateOptions{})
	return err
}

func (a *App) GetNamespaceYAML(name string) (string, error) {
	a.mu.RLock()
	cs := a.clients[a.activeContext]
	a.mu.RUnlock()
	if cs == nil {
		return "", fmt.Errorf("not connected")
	}

	ctx, cancel := context.WithTimeout(context.Background(), apiReadTimeout)
	defer cancel()
	ns, err := cs.CoreV1().Namespaces().Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		return "", fmt.Errorf("get Namespace: %w", err)
	}

	yamlBytes, err := sigsyaml.Marshal(ns)
	if err != nil {
		return "", fmt.Errorf("marshal Namespace to YAML: %w", err)
	}

	return string(yamlBytes), nil
}

func (a *App) UpdateNamespaceYAML(yamlString string) error {
	a.mu.RLock()
	cs := a.clients[a.activeContext]
	a.mu.RUnlock()
	if cs == nil {
		return fmt.Errorf("not connected")
	}

	var ns corev1.Namespace
	err := sigsyaml.Unmarshal([]byte(yamlString), &ns)
	if err != nil {
		return fmt.Errorf("unmarshal YAML to Namespace: %w", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), apiMutationTimeout)
	defer cancel()
	_, err = cs.CoreV1().Namespaces().Update(ctx, &ns, metav1.UpdateOptions{})
	if err != nil {
		return fmt.Errorf("update Namespace: %w", err)
	}

	a.emitNamespaces()

	return nil
}

package app

import (
	"context"
	"fmt"
	"log"
	"strings"

	"github.com/gknguyen/litelens/internal/dto"
	"github.com/gknguyen/litelens/internal/kube/resources"
	"github.com/wailsapp/wails/v2/pkg/runtime"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	sigsyaml "sigs.k8s.io/yaml"
)

func (a *App) GetServiceAccountByName(namespace, name string) (dto.ServiceAccount, error) {
	a.mu.RLock()
	h := a.factories[a.activeContext]
	a.mu.RUnlock()
	if h == nil {
		return dto.ServiceAccount{}, nil
	}
	if h.IsForbidden("serviceaccounts") {
		return dto.ServiceAccount{}, nil
	}
	<-h.GetSyncedChan("serviceaccounts")
	if h.IsForbidden("serviceaccounts") {
		return dto.ServiceAccount{}, nil
	}
	result, err := kubeResources.GetServiceAccountByName(
		h.Factory.Core().V1().ServiceAccounts().Lister(),
		namespace,
		name,
	)
	if err != nil {
		log.Printf("app: GetServiceAccountByName: %v", err)
		return dto.ServiceAccount{}, nil
	}
	return result, nil
}

func (a *App) ListServiceAccounts(namespace string) ([]dto.ServiceAccount, error) {
	a.mu.RLock()
	h := a.factories[a.activeContext]
	a.mu.RUnlock()
	if h == nil {
		return []dto.ServiceAccount{}, nil
	}
	if h.IsForbidden("serviceaccounts") {
		return []dto.ServiceAccount{}, nil
	}
	<-h.GetSyncedChan("serviceaccounts")
	if h.IsForbidden("serviceaccounts") {
		return nil, nil
	}
	result, err := kubeResources.ListServiceAccounts(
		h.Factory.Core().V1().ServiceAccounts().Lister(),
		namespace,
	)
	if err != nil {
		log.Printf("app: ListServiceAccounts: %v", err)
		return []dto.ServiceAccount{}, nil
	}
	return result, nil
}

func (a *App) DeleteServiceAccount(namespace, name string) error {
	a.mu.RLock()
	cs := a.clients[a.activeContext]
	a.mu.RUnlock()
	if cs == nil {
		return fmt.Errorf("not connected")
	}

	ctx, cancel := context.WithTimeout(context.Background(), apiMutationTimeout)
	defer cancel()
	err := cs.CoreV1().ServiceAccounts(namespace).Delete(ctx, name, metav1.DeleteOptions{})
	if err != nil && !errors.IsNotFound(err) {
		return fmt.Errorf("delete ServiceAccount: %w", err)
	}

	a.emitServiceAccounts(namespace)

	return nil
}

// DeleteServiceAccounts deletes multiple ServiceAccounts, handling best-effort deletion across namespaces.
func (a *App) DeleteServiceAccounts(items []dto.ServiceAccountRef) error {
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
		err := cs.CoreV1().ServiceAccounts(ref.Namespace).Delete(ctx, ref.Name, metav1.DeleteOptions{})
		cancel()
		if err != nil && !errors.IsNotFound(err) {
			msgs = append(msgs, fmt.Sprintf("%s/%s: %v", ref.Namespace, ref.Name, err))
		}
		namespaces[ref.Namespace] = true
	}

	// Emit updates for each unique namespace touched
	for ns := range namespaces {
		a.emitServiceAccounts(ns)
	}

	if len(msgs) > 0 {
		return fmt.Errorf("failed to delete %d of %d serviceaccounts: %s", len(msgs), len(items), strings.Join(msgs, "; "))
	}
	return nil
}

func (a *App) emitServiceAccounts(namespace string) {
	a.mu.RLock()
	h := a.factories[a.activeContext]
	a.mu.RUnlock()
	if h == nil {
		return
	}
	if h.IsForbidden("serviceaccounts") {
		return
	}
	<-h.GetSyncedChan("serviceaccounts")
	if h.IsForbidden("serviceaccounts") {
		return
	}
	lister := h.Factory.Core().V1().ServiceAccounts().Lister()
	allData, err := kubeResources.ListServiceAccounts(lister, "")
	if err != nil {
		log.Printf("app: emitServiceAccounts: %v", err)
		return
	}
	runtime.EventsEmit(a.ctx, "serviceaccounts:update", allData)
	if namespace != "" {
		nsData, err := kubeResources.ListServiceAccounts(lister, namespace)
		if err != nil {
			log.Printf("app: emitServiceAccounts ns=%s: %v", namespace, err)
			return
		}
		runtime.EventsEmit(a.ctx, "serviceaccounts:"+namespace+":update", nsData)
	}
}

func (a *App) GetServiceAccountYAML(namespace, name string) (string, error) {
	a.mu.RLock()
	cs := a.clients[a.activeContext]
	a.mu.RUnlock()
	if cs == nil {
		return "", fmt.Errorf("not connected")
	}

	ctx, cancel := context.WithTimeout(context.Background(), apiReadTimeout)
	defer cancel()
	sa, err := cs.CoreV1().ServiceAccounts(namespace).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		return "", fmt.Errorf("get ServiceAccount: %w", err)
	}

	yamlBytes, err := sigsyaml.Marshal(sa)
	if err != nil {
		return "", fmt.Errorf("marshal ServiceAccount to YAML: %w", err)
	}

	return string(yamlBytes), nil
}

func (a *App) UpdateServiceAccountYAML(namespace, yamlString string) error {
	a.mu.RLock()
	cs := a.clients[a.activeContext]
	a.mu.RUnlock()
	if cs == nil {
		return fmt.Errorf("not connected")
	}

	var sa corev1.ServiceAccount
	err := sigsyaml.Unmarshal([]byte(yamlString), &sa)
	if err != nil {
		return fmt.Errorf("unmarshal YAML to ServiceAccount: %w", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), apiMutationTimeout)
	defer cancel()
	_, err = cs.CoreV1().ServiceAccounts(namespace).Update(ctx, &sa, metav1.UpdateOptions{})
	if err != nil {
		return fmt.Errorf("update ServiceAccount: %w", err)
	}

	a.emitServiceAccounts(namespace)

	return nil
}

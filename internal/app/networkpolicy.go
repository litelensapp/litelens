package app

import (
	"context"
	"fmt"
	"log"
	"strings"

	"github.com/gknguyen/litelens/internal/dto"
	"github.com/gknguyen/litelens/internal/kube/resources"
	"github.com/wailsapp/wails/v2/pkg/runtime"
	networkingv1 "k8s.io/api/networking/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	sigsyaml "sigs.k8s.io/yaml"
)

func (a *App) ListNetworkPolicies(namespace string) ([]dto.NetworkPolicy, error) {
	a.mu.RLock()
	h := a.factories[a.activeContext]
	a.mu.RUnlock()
	if h == nil {
		return []dto.NetworkPolicy{}, nil
	}
	if h.IsForbidden("networkpolicies") {
		return []dto.NetworkPolicy{}, nil
	}
	<-h.GetSyncedChan("networkpolicies")
	if h.IsForbidden("networkpolicies") {
		return nil, nil
	}
	result, err := kubeResources.ListNetworkPolicies(h.Factory.Networking().V1().NetworkPolicies().Lister(), namespace)
	if err != nil {
		log.Printf("app: ListNetworkPolicies: %v", err)
		return []dto.NetworkPolicy{}, nil
	}
	return result, nil
}

func (a *App) GetNetworkPolicyByName(namespace, name string) (*dto.NetworkPolicyDetail, error) {
	a.mu.RLock()
	h := a.factories[a.activeContext]
	a.mu.RUnlock()
	if h == nil {
		return nil, nil
	}
	if h.IsForbidden("networkpolicies") {
		return nil, nil
	}
	<-h.GetSyncedChan("networkpolicies")
	if h.IsForbidden("networkpolicies") {
		return nil, nil
	}
	result, err := kubeResources.GetNetworkPolicyByName(h.Factory.Networking().V1().NetworkPolicies().Lister(), namespace, name)
	if err != nil {
		log.Printf("app: GetNetworkPolicyByName: %v", err)
		return nil, err
	}
	return result, nil
}

// DeleteNetworkPolicy deletes a NetworkPolicy from the specified namespace.
func (a *App) DeleteNetworkPolicy(namespace, name string) error {
	a.mu.RLock()
	cs := a.clients[a.activeContext]
	a.mu.RUnlock()
	if cs == nil {
		return fmt.Errorf("not connected")
	}

	ctx, cancel := context.WithTimeout(context.Background(), apiMutationTimeout)
	defer cancel()
	err := cs.NetworkingV1().NetworkPolicies(namespace).Delete(ctx, name, metav1.DeleteOptions{})
	if err != nil && !errors.IsNotFound(err) {
		return fmt.Errorf("delete NetworkPolicy: %w", err)
	}

	// Emit update event after successful delete
	a.emitNetworkPolicies(namespace)

	return nil
}

// DeleteNetworkPolicies deletes multiple NetworkPolicies, handling best-effort deletion across namespaces.
func (a *App) DeleteNetworkPolicies(items []dto.NetworkPolicyRef) error {
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
		err := cs.NetworkingV1().NetworkPolicies(ref.Namespace).Delete(ctx, ref.Name, metav1.DeleteOptions{})
		cancel()
		if err != nil && !errors.IsNotFound(err) {
			msgs = append(msgs, fmt.Sprintf("%s/%s: %v", ref.Namespace, ref.Name, err))
		}
		namespaces[ref.Namespace] = true
	}

	// Emit updates for each unique namespace touched
	for ns := range namespaces {
		a.emitNetworkPolicies(ns)
	}

	if len(msgs) > 0 {
		return fmt.Errorf("failed to delete %d of %d networkpolicies: %s", len(msgs), len(items), strings.Join(msgs, "; "))
	}
	return nil
}

func (a *App) emitNetworkPolicies(namespace string) {
	a.mu.RLock()
	h := a.factories[a.activeContext]
	a.mu.RUnlock()
	if h == nil {
		return
	}
	if h.IsForbidden("networkpolicies") {
		return
	}
	<-h.GetSyncedChan("networkpolicies")
	if h.IsForbidden("networkpolicies") {
		return
	}
	lister := h.Factory.Networking().V1().NetworkPolicies().Lister()
	allData, err := kubeResources.ListNetworkPolicies(lister, "")
	if err != nil {
		log.Printf("app: emitNetworkPolicies: %v", err)
		return
	}
	runtime.EventsEmit(a.ctx, "networkpolicies:update", allData)
	if namespace != "" {
		nsData, err := kubeResources.ListNetworkPolicies(lister, namespace)
		if err != nil {
			log.Printf("app: emitNetworkPolicies ns=%s: %v", namespace, err)
			return
		}
		runtime.EventsEmit(a.ctx, "networkpolicies:"+namespace+":update", nsData)
	}
}

func (a *App) GetNetworkPolicyYAML(namespace, name string) (string, error) {
	a.mu.RLock()
	cs := a.clients[a.activeContext]
	a.mu.RUnlock()
	if cs == nil {
		return "", fmt.Errorf("not connected")
	}

	ctx, cancel := context.WithTimeout(context.Background(), apiReadTimeout)
	defer cancel()
	np, err := cs.NetworkingV1().NetworkPolicies(namespace).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		return "", fmt.Errorf("get NetworkPolicy: %w", err)
	}

	b, err := sigsyaml.Marshal(np)
	if err != nil {
		return "", fmt.Errorf("marshal NetworkPolicy to YAML: %w", err)
	}

	return string(b), nil
}

func (a *App) UpdateNetworkPolicyYAML(namespace, yamlString string) error {
	a.mu.RLock()
	cs := a.clients[a.activeContext]
	a.mu.RUnlock()
	if cs == nil {
		return fmt.Errorf("not connected")
	}

	var np networkingv1.NetworkPolicy
	err := sigsyaml.Unmarshal([]byte(yamlString), &np)
	if err != nil {
		return fmt.Errorf("unmarshal YAML to NetworkPolicy: %w", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), apiMutationTimeout)
	defer cancel()
	_, err = cs.NetworkingV1().NetworkPolicies(namespace).Update(ctx, &np, metav1.UpdateOptions{})
	if err != nil {
		return fmt.Errorf("update NetworkPolicy: %w", err)
	}

	a.emitNetworkPolicies(namespace)

	return nil
}

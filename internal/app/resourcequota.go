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
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	sigsyaml "sigs.k8s.io/yaml"
)

func (a *App) ListResourceQuotas(namespace string) ([]dto.ResourceQuota, error) {
	a.mu.RLock()
	h := a.factories[a.activeContext]
	a.mu.RUnlock()
	if h == nil {
		return []dto.ResourceQuota{}, nil
	}
	if h.IsForbidden("resourcequotas") {
		return []dto.ResourceQuota{}, nil
	}
	<-h.GetSyncedChan("resourcequotas")
	if h.IsForbidden("resourcequotas") {
		return nil, nil
	}
	result, err := kubeResources.ListResourceQuotas(h.Factory.Core().V1().ResourceQuotas().Lister(), namespace)
	if err != nil {
		log.Printf("app: ListResourceQuotas: %v", err)
		return []dto.ResourceQuota{}, nil
	}
	return result, nil
}

func (a *App) GetResourceQuotaByName(namespace, name string) (dto.ResourceQuotaDetail, error) {
	a.mu.RLock()
	h := a.factories[a.activeContext]
	a.mu.RUnlock()
	if h == nil {
		return dto.ResourceQuotaDetail{}, nil
	}
	if h.IsForbidden("resourcequotas") {
		return dto.ResourceQuotaDetail{}, nil
	}
	<-h.GetSyncedChan("resourcequotas")
	if h.IsForbidden("resourcequotas") {
		return dto.ResourceQuotaDetail{}, nil
	}
	result, err := kubeResources.GetResourceQuotaByName(h.Factory.Core().V1().ResourceQuotas().Lister(), namespace, name)
	if err != nil {
		log.Printf("app: GetResourceQuotaByName: %v", err)
		return dto.ResourceQuotaDetail{}, nil
	}
	return result, nil
}

func (a *App) CreateResourceQuota(namespace, name string, hard map[string]string) error {
	a.mu.RLock()
	cs := a.clients[a.activeContext]
	a.mu.RUnlock()
	if cs == nil {
		return fmt.Errorf("not connected")
	}

	resourceList := make(corev1.ResourceList)
	for k, v := range hard {
		qty, err := resource.ParseQuantity(v)
		if err != nil {
			return fmt.Errorf("invalid quantity for %q: %w", k, err)
		}
		resourceList[corev1.ResourceName(k)] = qty
	}

	rq := &corev1.ResourceQuota{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		Spec: corev1.ResourceQuotaSpec{
			Hard: resourceList,
		},
	}

	ctx, cancel := context.WithTimeout(context.Background(), apiMutationTimeout)
	defer cancel()
	_, err := cs.CoreV1().ResourceQuotas(namespace).Create(ctx, rq, metav1.CreateOptions{})
	return err
}

// DeleteResourceQuota deletes a ResourceQuota from the specified namespace.
func (a *App) DeleteResourceQuota(namespace, name string) error {
	a.mu.RLock()
	cs := a.clients[a.activeContext]
	a.mu.RUnlock()
	if cs == nil {
		return fmt.Errorf("not connected")
	}

	ctx, cancel := context.WithTimeout(context.Background(), apiMutationTimeout)
	defer cancel()
	err := cs.CoreV1().ResourceQuotas(namespace).Delete(ctx, name, metav1.DeleteOptions{})
	if err != nil && !errors.IsNotFound(err) {
		return fmt.Errorf("delete ResourceQuota: %w", err)
	}

	// Emit update event after successful delete
	a.emitResourceQuotas(namespace)

	return nil
}

// DeleteResourceQuotas deletes multiple ResourceQuotas, handling best-effort deletion across namespaces.
func (a *App) DeleteResourceQuotas(items []dto.ResourceQuotaRef) error {
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
		err := cs.CoreV1().ResourceQuotas(ref.Namespace).Delete(ctx, ref.Name, metav1.DeleteOptions{})
		cancel()
		if err != nil && !errors.IsNotFound(err) {
			msgs = append(msgs, fmt.Sprintf("%s/%s: %v", ref.Namespace, ref.Name, err))
		}
		namespaces[ref.Namespace] = true
	}

	// Emit updates for each unique namespace touched
	for ns := range namespaces {
		a.emitResourceQuotas(ns)
	}

	if len(msgs) > 0 {
		return fmt.Errorf("failed to delete %d of %d resourcequotas: %s", len(msgs), len(items), strings.Join(msgs, "; "))
	}
	return nil
}

func (a *App) emitResourceQuotas(namespace string) {
	a.mu.RLock()
	h := a.factories[a.activeContext]
	a.mu.RUnlock()
	if h == nil {
		return
	}
	if h.IsForbidden("resourcequotas") {
		return
	}
	<-h.GetSyncedChan("resourcequotas")
	if h.IsForbidden("resourcequotas") {
		return
	}
	lister := h.Factory.Core().V1().ResourceQuotas().Lister()
	allData, err := kubeResources.ListResourceQuotas(lister, "")
	if err != nil {
		log.Printf("app: emitResourceQuotas: %v", err)
		return
	}
	runtime.EventsEmit(a.ctx, "resourcequotas:update", allData)
	if namespace != "" {
		// Filter already-fetched cluster-wide data instead of re-listing
		nsData := make([]dto.ResourceQuota, 0)
		for _, item := range allData {
			if item.Namespace == namespace {
				nsData = append(nsData, item)
			}
		}
		runtime.EventsEmit(a.ctx, "resourcequotas:"+namespace+":update", nsData)
	}
}

func (a *App) GetResourceQuotaYAML(namespace, name string) (string, error) {
	a.mu.RLock()
	cs := a.clients[a.activeContext]
	a.mu.RUnlock()
	if cs == nil {
		return "", fmt.Errorf("not connected")
	}

	ctx, cancel := context.WithTimeout(context.Background(), apiReadTimeout)
	defer cancel()
	rq, err := cs.CoreV1().ResourceQuotas(namespace).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		return "", fmt.Errorf("get ResourceQuota: %w", err)
	}

	b, err := sigsyaml.Marshal(rq)
	if err != nil {
		return "", fmt.Errorf("marshal ResourceQuota to YAML: %w", err)
	}

	return string(b), nil
}

func (a *App) UpdateResourceQuotaYAML(namespace, yamlString string) error {
	a.mu.RLock()
	cs := a.clients[a.activeContext]
	a.mu.RUnlock()
	if cs == nil {
		return fmt.Errorf("not connected")
	}

	var rq corev1.ResourceQuota
	err := sigsyaml.Unmarshal([]byte(yamlString), &rq)
	if err != nil {
		return fmt.Errorf("unmarshal YAML to ResourceQuota: %w", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), apiMutationTimeout)
	defer cancel()
	_, err = cs.CoreV1().ResourceQuotas(namespace).Update(ctx, &rq, metav1.UpdateOptions{})
	if err != nil {
		return fmt.Errorf("update ResourceQuota: %w", err)
	}

	a.emitResourceQuotas(namespace)

	return nil
}

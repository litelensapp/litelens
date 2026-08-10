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
	resource_api "k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	sigsyaml "sigs.k8s.io/yaml"
)

func (a *App) ListLimitRanges(namespace string) ([]dto.LimitRange, error) {
	a.mu.RLock()
	h := a.factories[a.activeContext]
	a.mu.RUnlock()
	if h == nil {
		return []dto.LimitRange{}, nil
	}
	if h.IsForbidden("limitranges") {
		return []dto.LimitRange{}, nil
	}
	<-h.GetSyncedChan("limitranges")
	if h.IsForbidden("limitranges") {
		return []dto.LimitRange{}, nil
	}
	result, err := kubeResources.ListLimitRanges(h.Factory.Core().V1().LimitRanges().Lister(), namespace)
	if err != nil {
		log.Printf("app: ListLimitRanges: %v", err)
		return []dto.LimitRange{}, nil
	}
	return result, nil
}

func (a *App) GetLimitRangeByName(namespace, name string) dto.LimitRangeDetail {
	a.mu.RLock()
	h := a.factories[a.activeContext]
	a.mu.RUnlock()
	if h == nil {
		return dto.LimitRangeDetail{}
	}
	if h.IsForbidden("limitranges") {
		return dto.LimitRangeDetail{}
	}
	<-h.GetSyncedChan("limitranges")
	if h.IsForbidden("limitranges") {
		return dto.LimitRangeDetail{}
	}
	result, err := kubeResources.GetLimitRangeByName(h.Factory.Core().V1().LimitRanges().Lister(), namespace, name)
	if err != nil {
		log.Printf("app: GetLimitRangeByName: %v", err)
		return dto.LimitRangeDetail{}
	}
	return result
}

// CreateLimitRange creates a new LimitRange in the given namespace.
// limits format: map[limitType]map["resource/valueType"]value
// limitType is one of: "Container", "Pod", "PersistentVolumeClaim"
// e.g., { "Container": {"cpu/Max": "1", "memory/Default": "256Mi"}, "Pod": {"cpu/Max": "2"} }
func (a *App) CreateLimitRange(namespace, name string, limits map[string]map[string]string) error {
	a.mu.RLock()
	cs := a.clients[a.activeContext]
	a.mu.RUnlock()
	if cs == nil {
		return fmt.Errorf("not connected")
	}

	limitTypeMap := map[string]corev1.LimitType{
		"Container":             corev1.LimitTypeContainer,
		"Pod":                   corev1.LimitTypePod,
		"PersistentVolumeClaim": corev1.LimitTypePersistentVolumeClaim,
	}

	// Build one LimitRangeItem per limitType — Kubernetes rejects duplicate types.
	itemMap := map[string]*corev1.LimitRangeItem{}
	for limitType, resourceValues := range limits {
		lrt, ok := limitTypeMap[limitType]
		if !ok {
			return fmt.Errorf("unknown limit type %q", limitType)
		}
		item, exists := itemMap[limitType]
		if !exists {
			item = &corev1.LimitRangeItem{Type: lrt}
			itemMap[limitType] = item
		}
		for key, val := range resourceValues {
			// key format: "resource/valueType" e.g., "cpu/Max", "memory/Default"
			parts := strings.SplitN(key, "/", 2)
			if len(parts) != 2 {
				continue
			}
			resource := corev1.ResourceName(parts[0])
			valueType := parts[1]
			qty, err := resource_api.ParseQuantity(val)
			if err != nil {
				return fmt.Errorf("invalid quantity %q for %s: %w", val, key, err)
			}
			switch valueType {
			case "Min":
				if item.Min == nil {
					item.Min = corev1.ResourceList{}
				}
				item.Min[resource] = qty
			case "Max":
				if item.Max == nil {
					item.Max = corev1.ResourceList{}
				}
				item.Max[resource] = qty
			case "Default":
				if item.Default == nil {
					item.Default = corev1.ResourceList{}
				}
				item.Default[resource] = qty
			case "DefaultRequest":
				if item.DefaultRequest == nil {
					item.DefaultRequest = corev1.ResourceList{}
				}
				item.DefaultRequest[resource] = qty
			}
		}
	}
	items := make([]corev1.LimitRangeItem, 0, len(itemMap))
	for _, item := range itemMap {
		items = append(items, *item)
	}

	lr := &corev1.LimitRange{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		Spec: corev1.LimitRangeSpec{
			Limits: items,
		},
	}

	ctx, cancel := context.WithTimeout(context.Background(), apiMutationTimeout)
	defer cancel()
	_, err := cs.CoreV1().LimitRanges(namespace).Create(ctx, lr, metav1.CreateOptions{})
	if err != nil {
		return err
	}

	a.emitLimitRanges(namespace)
	return nil
}

func (a *App) DeleteLimitRange(namespace, name string) error {
	a.mu.RLock()
	cs := a.clients[a.activeContext]
	a.mu.RUnlock()
	if cs == nil {
		return fmt.Errorf("not connected")
	}

	ctx, cancel := context.WithTimeout(context.Background(), apiMutationTimeout)
	defer cancel()
	err := cs.CoreV1().LimitRanges(namespace).Delete(ctx, name, metav1.DeleteOptions{})
	if err != nil && !errors.IsNotFound(err) {
		return fmt.Errorf("delete LimitRange: %w", err)
	}

	a.emitLimitRanges(namespace)

	return nil
}

// DeleteLimitRanges deletes multiple LimitRanges, handling best-effort deletion across namespaces.
func (a *App) DeleteLimitRanges(items []dto.LimitRangeRef) error {
	a.mu.RLock()
	cs := a.clients[a.activeContext]
	a.mu.RUnlock()
	if cs == nil {
		return fmt.Errorf("not connected")
	}

	var failMsgs []string
	touchedNamespaces := map[string]bool{}

	for _, ref := range items {
		ctx, cancel := context.WithTimeout(context.Background(), apiMutationTimeout)
		err := cs.CoreV1().LimitRanges(ref.Namespace).Delete(ctx, ref.Name, metav1.DeleteOptions{})
		cancel()
		if err != nil && !errors.IsNotFound(err) {
			failMsgs = append(failMsgs, fmt.Sprintf("%s/%s: %v", ref.Namespace, ref.Name, err))
		}
		touchedNamespaces[ref.Namespace] = true
	}

	for ns := range touchedNamespaces {
		a.emitLimitRanges(ns)
	}

	if len(failMsgs) > 0 {
		return fmt.Errorf("failed to delete %d of %d limitranges: %s", len(failMsgs), len(items), strings.Join(failMsgs, "; "))
	}
	return nil
}

func (a *App) emitLimitRanges(namespace string) {
	a.mu.RLock()
	h := a.factories[a.activeContext]
	a.mu.RUnlock()
	if h == nil {
		return
	}
	if h.IsForbidden("limitranges") {
		return
	}
	<-h.GetSyncedChan("limitranges")
	if h.IsForbidden("limitranges") {
		return
	}
	lister := h.Factory.Core().V1().LimitRanges().Lister()
	allData, err := kubeResources.ListLimitRanges(lister, "")
	if err != nil {
		log.Printf("app: emitLimitRanges: %v", err)
		return
	}
	runtime.EventsEmit(a.ctx, "limitranges:update", allData)
	if namespace != "" {
		nsData, err := kubeResources.ListLimitRanges(lister, namespace)
		if err != nil {
			log.Printf("app: emitLimitRanges ns=%s: %v", namespace, err)
			return
		}
		runtime.EventsEmit(a.ctx, "limitranges:"+namespace+":update", nsData)
	}
}

func (a *App) GetLimitRangeYAML(namespace, name string) (string, error) {
	a.mu.RLock()
	cs := a.clients[a.activeContext]
	a.mu.RUnlock()
	if cs == nil {
		return "", fmt.Errorf("not connected")
	}

	ctx, cancel := context.WithTimeout(context.Background(), apiReadTimeout)
	defer cancel()
	lr, err := cs.CoreV1().LimitRanges(namespace).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		return "", fmt.Errorf("get LimitRange: %w", err)
	}

	b, err := sigsyaml.Marshal(lr)
	if err != nil {
		return "", fmt.Errorf("marshal LimitRange: %w", err)
	}

	return string(b), nil
}

func (a *App) UpdateLimitRangeYAML(namespace, yamlString string) error {
	a.mu.RLock()
	cs := a.clients[a.activeContext]
	a.mu.RUnlock()
	if cs == nil {
		return fmt.Errorf("not connected")
	}

	var lr corev1.LimitRange
	err := sigsyaml.Unmarshal([]byte(yamlString), &lr)
	if err != nil {
		return fmt.Errorf("unmarshal YAML to LimitRange: %w", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), apiMutationTimeout)
	defer cancel()
	_, err = cs.CoreV1().LimitRanges(namespace).Update(ctx, &lr, metav1.UpdateOptions{})
	if err != nil {
		return fmt.Errorf("update LimitRange: %w", err)
	}

	a.emitLimitRanges(namespace)

	return nil
}

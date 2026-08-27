package app

import (
	"context"
	"fmt"
	"log"
	"strings"

	kubeResources "github.com/litelensapp/litelens/internal/kube/resources"
	"github.com/litelensapp/litelens/packages/core/kube/dto"
	"github.com/wailsapp/wails/v2/pkg/runtime"
	appsv1 "k8s.io/api/apps/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	sigsyaml "sigs.k8s.io/yaml"
)

func (a *App) GetStatefulSetByName(namespace, name string) (dto.StatefulSet, error) {
	a.mu.RLock()
	h := a.factories[a.activeContext]
	a.mu.RUnlock()
	if h == nil {
		return dto.StatefulSet{}, nil
	}
	if h.IsForbidden("statefulsets") {
		return dto.StatefulSet{}, nil
	}
	<-h.GetSyncedChan("statefulsets")
	if h.IsForbidden("statefulsets") {
		return dto.StatefulSet{}, nil
	}
	result, err := kubeResources.GetStatefulSetByName(h.Factory.Apps().V1().StatefulSets().Lister(), namespace, name)
	if err != nil {
		return dto.StatefulSet{}, err
	}
	return result, nil
}

func (a *App) ListStatefulSets() ([]dto.StatefulSet, error) {
	a.mu.RLock()
	h := a.factories[a.activeContext]
	namespaces := a.activeNamespaces
	a.mu.RUnlock()
	if h == nil {
		return []dto.StatefulSet{}, nil
	}
	if h.IsForbidden("statefulsets") {
		return []dto.StatefulSet{}, nil
	}
	<-h.GetSyncedChan("statefulsets")
	if h.IsForbidden("statefulsets") {
		return nil, nil
	}
	result, err := kubeResources.ListStatefulSets(h.Factory.Apps().V1().StatefulSets().Lister(), namespaces)
	if err != nil {
		log.Printf("app: ListStatefulSets: %v", err)
		return []dto.StatefulSet{}, nil
	}
	return result, nil
}

func (a *App) GetStatefulSetsSummary() (dto.StatefulSetSummary, error) {
	a.mu.RLock()
	h := a.factories[a.activeContext]
	namespaces := a.activeNamespaces
	a.mu.RUnlock()
	if h == nil {
		return dto.StatefulSetSummary{}, nil
	}
	if h.IsForbidden("statefulsets") {
		return dto.StatefulSetSummary{}, nil
	}
	<-h.GetSyncedChan("statefulsets")
	if h.IsForbidden("statefulsets") {
		return dto.StatefulSetSummary{}, nil
	}
	lister := h.Factory.Apps().V1().StatefulSets().Lister()
	sss, err := lister.List(labels.Everything())
	if err != nil {
		log.Printf("app: GetStatefulSetsSummary: %v", err)
		return dto.StatefulSetSummary{}, nil
	}
	if len(namespaces) > 0 {
		nsSet := make(map[string]struct{}, len(namespaces))
		for _, ns := range namespaces {
			nsSet[ns] = struct{}{}
		}
		filtered := sss[:0:0]
		for _, ss := range sss {
			if _, ok := nsSet[ss.Namespace]; ok {
				filtered = append(filtered, ss)
			}
		}
		sss = filtered
	}
	return kubeResources.SummarizeStatefulSets(sss), nil
}

// DeleteStatefulSet deletes a StatefulSet from the specified namespace.
func (a *App) DeleteStatefulSet(namespace, name string) error {
	a.mu.RLock()
	cs := a.clients[a.activeContext]
	a.mu.RUnlock()
	if cs == nil {
		return fmt.Errorf("not connected")
	}

	ctx, cancel := context.WithTimeout(context.Background(), apiMutationTimeout)
	defer cancel()
	err := cs.AppsV1().StatefulSets(namespace).Delete(ctx, name, metav1.DeleteOptions{})
	if err != nil && !errors.IsNotFound(err) {
		return fmt.Errorf("delete StatefulSet: %w", err)
	}

	// Emit update event after successful delete
	a.emitStatefulSets()

	return nil
}

// DeleteStatefulSets deletes multiple StatefulSets, handling best-effort deletion across namespaces.
func (a *App) DeleteStatefulSets(items []dto.StatefulSetRef) error {
	a.mu.RLock()
	cs := a.clients[a.activeContext]
	a.mu.RUnlock()
	if cs == nil {
		return fmt.Errorf("not connected")
	}

	var msgs []string

	for _, ref := range items {
		ctx, cancel := context.WithTimeout(context.Background(), apiMutationTimeout)
		err := cs.AppsV1().StatefulSets(ref.Namespace).Delete(ctx, ref.Name, metav1.DeleteOptions{})
		cancel()
		if err != nil && !errors.IsNotFound(err) {
			msgs = append(msgs, fmt.Sprintf("%s/%s: %v", ref.Namespace, ref.Name, err))
		}
	}

	a.emitStatefulSets()

	if len(msgs) > 0 {
		return fmt.Errorf("failed to delete %d of %d statefulsets: %s", len(msgs), len(items), strings.Join(msgs, "; "))
	}
	return nil
}

func (a *App) emitStatefulSets() {
	a.mu.RLock()
	h := a.factories[a.activeContext]
	namespaces := a.activeNamespaces
	a.mu.RUnlock()
	if h == nil {
		return
	}
	if h.IsForbidden("statefulsets") {
		return
	}
	<-h.GetSyncedChan("statefulsets")
	if h.IsForbidden("statefulsets") {
		return
	}
	lister := h.Factory.Apps().V1().StatefulSets().Lister()
	data, err := kubeResources.ListStatefulSets(lister, namespaces)
	if err != nil {
		log.Printf("app: emitStatefulSets: %v", err)
		return
	}
	runtime.EventsEmit(a.ctx, "statefulsets:update", data)
}

func (a *App) GetStatefulSetYAML(namespace, name string) (string, error) {
	a.mu.RLock()
	cs := a.clients[a.activeContext]
	a.mu.RUnlock()
	if cs == nil {
		return "", fmt.Errorf("not connected")
	}

	ctx, cancel := context.WithTimeout(context.Background(), apiReadTimeout)
	defer cancel()
	ss, err := cs.AppsV1().StatefulSets(namespace).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		return "", fmt.Errorf("get StatefulSet: %w", err)
	}

	yamlBytes, err := sigsyaml.Marshal(ss)
	if err != nil {
		return "", fmt.Errorf("marshal StatefulSet to YAML: %w", err)
	}

	return string(yamlBytes), nil
}

func (a *App) UpdateStatefulSetYAML(namespace, yamlString string) error {
	a.mu.RLock()
	cs := a.clients[a.activeContext]
	a.mu.RUnlock()
	if cs == nil {
		return fmt.Errorf("not connected")
	}

	var ss appsv1.StatefulSet
	err := sigsyaml.Unmarshal([]byte(yamlString), &ss)
	if err != nil {
		return fmt.Errorf("unmarshal YAML to StatefulSet: %w", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), apiMutationTimeout)
	defer cancel()
	_, err = cs.AppsV1().StatefulSets(namespace).Update(ctx, &ss, metav1.UpdateOptions{})
	if err != nil {
		return fmt.Errorf("update StatefulSet: %w", err)
	}

	a.emitStatefulSets()

	return nil
}

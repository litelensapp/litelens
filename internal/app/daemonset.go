package app

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/litelensapp/litelens/packages/core/dto"
	kubeResources "github.com/litelensapp/litelens/internal/kube/resources"
	"github.com/wailsapp/wails/v2/pkg/runtime"
	appsv1 "k8s.io/api/apps/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/types"
	sigsyaml "sigs.k8s.io/yaml"
)

func (a *App) ListDaemonSets(namespace string) ([]dto.DaemonSet, error) {
	a.mu.RLock()
	h := a.factories[a.activeContext]
	a.mu.RUnlock()
	if h == nil {
		return []dto.DaemonSet{}, nil
	}
	if h.IsForbidden("daemonsets") {
		return []dto.DaemonSet{}, nil
	}
	<-h.GetSyncedChan("daemonsets")
	if h.IsForbidden("daemonsets") {
		return nil, nil
	}
	result, err := kubeResources.ListDaemonSets(h.Factory.Apps().V1().DaemonSets().Lister(), namespace)
	if err != nil {
		log.Printf("app: ListDaemonSets: %v", err)
		return []dto.DaemonSet{}, nil
	}
	return result, nil
}

func (a *App) GetDaemonSetByName(namespace, name string) (dto.DaemonSet, error) {
	a.mu.RLock()
	h := a.factories[a.activeContext]
	a.mu.RUnlock()
	if h == nil {
		return dto.DaemonSet{}, nil
	}
	if h.IsForbidden("daemonsets") {
		return dto.DaemonSet{}, nil
	}
	<-h.GetSyncedChan("daemonsets")
	if h.IsForbidden("daemonsets") {
		return dto.DaemonSet{}, nil
	}
	result, err := kubeResources.GetDaemonSetByName(h.Factory.Apps().V1().DaemonSets().Lister(), namespace, name)
	if err != nil {
		log.Printf("app: GetDaemonSetByName: %v", err)
		return dto.DaemonSet{}, nil
	}
	return result, nil
}

func (a *App) GetDaemonSetsSummary(namespace string) (dto.DaemonSetSummary, error) {
	a.mu.RLock()
	h := a.factories[a.activeContext]
	a.mu.RUnlock()
	if h == nil {
		return dto.DaemonSetSummary{}, nil
	}
	if h.IsForbidden("daemonsets") {
		return dto.DaemonSetSummary{}, nil
	}
	<-h.GetSyncedChan("daemonsets")
	if h.IsForbidden("daemonsets") {
		return dto.DaemonSetSummary{}, nil
	}
	var dss []*appsv1.DaemonSet
	var err error
	lister := h.Factory.Apps().V1().DaemonSets().Lister()
	if namespace == "" {
		dss, err = lister.List(labels.Everything())
	} else {
		dss, err = lister.DaemonSets(namespace).List(labels.Everything())
	}
	if err != nil {
		log.Printf("app: GetDaemonSetsSummary: %v", err)
		return dto.DaemonSetSummary{}, nil
	}
	return kubeResources.SummarizeDaemonSets(dss), nil
}

func (a *App) RestartDaemonSet(namespace, name string) error {
	a.mu.RLock()
	cs := a.clients[a.activeContext]
	a.mu.RUnlock()
	if cs == nil {
		return fmt.Errorf("not connected")
	}
	patchBody := map[string]any{
		"spec": map[string]any{
			"template": map[string]any{
				"metadata": map[string]any{
					"annotations": map[string]any{
						"kubectl.kubernetes.io/restartedAt": time.Now().UTC().Format(time.RFC3339),
					},
				},
			},
		},
	}
	patchBytes, err := json.Marshal(patchBody)
	if err != nil {
		return err
	}
	ctx, cancel := context.WithTimeout(context.Background(), apiMutationTimeout)
	defer cancel()
	_, err = cs.AppsV1().DaemonSets(namespace).Patch(
		ctx, name, types.MergePatchType, patchBytes, metav1.PatchOptions{},
	)
	return err
}

func (a *App) DeleteDaemonSet(namespace, name string) error {
	a.mu.RLock()
	cs := a.clients[a.activeContext]
	a.mu.RUnlock()
	if cs == nil {
		return fmt.Errorf("not connected")
	}

	ctx, cancel := context.WithTimeout(context.Background(), apiMutationTimeout)
	defer cancel()
	err := cs.AppsV1().DaemonSets(namespace).Delete(ctx, name, metav1.DeleteOptions{})
	if err != nil && !errors.IsNotFound(err) {
		return fmt.Errorf("delete DaemonSet: %w", err)
	}

	a.emitDaemonSets(namespace)

	return nil
}

// DeleteDaemonSets deletes multiple DaemonSets, handling best-effort deletion across namespaces.
func (a *App) DeleteDaemonSets(items []dto.DaemonSetRef) error {
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
		err := cs.AppsV1().DaemonSets(ref.Namespace).Delete(ctx, ref.Name, metav1.DeleteOptions{})
		cancel()
		if err != nil && !errors.IsNotFound(err) {
			msgs = append(msgs, fmt.Sprintf("%s/%s: %v", ref.Namespace, ref.Name, err))
		}
		namespaces[ref.Namespace] = true
	}

	// Emit updates for each unique namespace touched
	for ns := range namespaces {
		a.emitDaemonSets(ns)
	}

	if len(msgs) > 0 {
		return fmt.Errorf("failed to delete %d of %d daemonsets: %s", len(msgs), len(items), strings.Join(msgs, "; "))
	}
	return nil
}

func (a *App) emitDaemonSets(namespace string) {
	a.mu.RLock()
	h := a.factories[a.activeContext]
	a.mu.RUnlock()
	if h == nil {
		return
	}
	if h.IsForbidden("daemonsets") {
		return
	}
	<-h.GetSyncedChan("daemonsets")
	if h.IsForbidden("daemonsets") {
		return
	}
	lister := h.Factory.Apps().V1().DaemonSets().Lister()
	allData, err := kubeResources.ListDaemonSets(lister, "")
	if err != nil {
		log.Printf("app: emitDaemonSets: %v", err)
		return
	}
	runtime.EventsEmit(a.ctx, "daemonsets:update", allData)
	if namespace != "" {
		// Filter already-fetched cluster-wide data instead of re-listing
		nsData := make([]dto.DaemonSet, 0)
		for _, ds := range allData {
			if ds.Namespace == namespace {
				nsData = append(nsData, ds)
			}
		}
		runtime.EventsEmit(a.ctx, "daemonsets:"+namespace+":update", nsData)
	}
}

func (a *App) GetDaemonSetYAML(namespace, name string) (string, error) {
	a.mu.RLock()
	cs := a.clients[a.activeContext]
	a.mu.RUnlock()
	if cs == nil {
		return "", fmt.Errorf("not connected")
	}

	ctx, cancel := context.WithTimeout(context.Background(), apiReadTimeout)
	defer cancel()
	ds, err := cs.AppsV1().DaemonSets(namespace).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		return "", fmt.Errorf("get DaemonSet: %w", err)
	}

	yamlBytes, err := sigsyaml.Marshal(ds)
	if err != nil {
		return "", fmt.Errorf("marshal DaemonSet to YAML: %w", err)
	}

	return string(yamlBytes), nil
}

func (a *App) UpdateDaemonSetYAML(namespace, yamlString string) error {
	a.mu.RLock()
	cs := a.clients[a.activeContext]
	a.mu.RUnlock()
	if cs == nil {
		return fmt.Errorf("not connected")
	}

	var ds appsv1.DaemonSet
	err := sigsyaml.Unmarshal([]byte(yamlString), &ds)
	if err != nil {
		return fmt.Errorf("unmarshal YAML to DaemonSet: %w", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), apiMutationTimeout)
	defer cancel()
	_, err = cs.AppsV1().DaemonSets(namespace).Update(ctx, &ds, metav1.UpdateOptions{})
	if err != nil {
		return fmt.Errorf("update DaemonSet: %w", err)
	}

	a.emitDaemonSets(namespace)

	return nil
}

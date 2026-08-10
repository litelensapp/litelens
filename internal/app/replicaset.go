package app

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"strings"

	"github.com/litelensapp/litelens/internal/dto"
	"github.com/litelensapp/litelens/internal/kube/resources"
	"github.com/wailsapp/wails/v2/pkg/runtime"
	appsv1 "k8s.io/api/apps/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	sigsyaml "sigs.k8s.io/yaml"
)

func (a *App) ListReplicaSets(namespace string) ([]dto.ReplicaSet, error) {
	a.mu.RLock()
	h := a.factories[a.activeContext]
	a.mu.RUnlock()
	if h == nil {
		return []dto.ReplicaSet{}, nil
	}
	if h.IsForbidden("replicasets") {
		return []dto.ReplicaSet{}, nil
	}
	<-h.GetSyncedChan("replicasets")
	if h.IsForbidden("replicasets") {
		return nil, nil
	}
	result, err := kubeResources.ListReplicaSets(h.Factory.Apps().V1().ReplicaSets().Lister(), namespace)
	if err != nil {
		log.Printf("app: ListReplicaSets: %v", err)
		return []dto.ReplicaSet{}, nil
	}
	return result, nil
}

func (a *App) GetReplicaSetByName(namespace, name string) (dto.ReplicaSet, error) {
	a.mu.RLock()
	h := a.factories[a.activeContext]
	a.mu.RUnlock()
	if h == nil {
		return dto.ReplicaSet{}, nil
	}
	if h.IsForbidden("replicasets") {
		return dto.ReplicaSet{}, nil
	}
	<-h.GetSyncedChan("replicasets")
	if h.IsForbidden("replicasets") {
		return dto.ReplicaSet{}, nil
	}
	result, err := kubeResources.GetReplicaSetByName(h.Factory.Apps().V1().ReplicaSets().Lister(), namespace, name)
	if err != nil {
		log.Printf("app: GetReplicaSetByName: %v", err)
		return dto.ReplicaSet{}, nil
	}
	return result, nil
}

func (a *App) ScaleReplicaSet(namespace, name string, replicas int32) error {
	a.mu.RLock()
	cs := a.clients[a.activeContext]
	a.mu.RUnlock()
	if cs == nil {
		return fmt.Errorf("not connected")
	}
	patchBody := map[string]any{
		"spec": map[string]any{"replicas": replicas},
	}
	patchBytes, err := json.Marshal(patchBody)
	if err != nil {
		return err
	}
	ctx, cancel := context.WithTimeout(context.Background(), apiMutationTimeout)
	defer cancel()
	_, err = cs.AppsV1().ReplicaSets(namespace).Patch(
		ctx, name, types.MergePatchType, patchBytes, metav1.PatchOptions{},
	)
	return err
}

func (a *App) DeleteReplicaSet(namespace, name string) error {
	a.mu.RLock()
	cs := a.clients[a.activeContext]
	a.mu.RUnlock()
	if cs == nil {
		return fmt.Errorf("not connected")
	}

	ctx, cancel := context.WithTimeout(context.Background(), apiMutationTimeout)
	defer cancel()
	err := cs.AppsV1().ReplicaSets(namespace).Delete(ctx, name, metav1.DeleteOptions{})
	if err != nil && !errors.IsNotFound(err) {
		return fmt.Errorf("delete ReplicaSet: %w", err)
	}

	a.emitReplicaSets(namespace)

	return nil
}

// DeleteReplicaSets deletes multiple ReplicaSets, handling best-effort deletion across namespaces.
func (a *App) DeleteReplicaSets(items []dto.ReplicaSetRef) error {
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
		err := cs.AppsV1().ReplicaSets(ref.Namespace).Delete(ctx, ref.Name, metav1.DeleteOptions{})
		cancel()
		if err != nil && !errors.IsNotFound(err) {
			msgs = append(msgs, fmt.Sprintf("%s/%s: %v", ref.Namespace, ref.Name, err))
		}
		namespaces[ref.Namespace] = true
	}

	// Emit updates for each unique namespace touched
	for ns := range namespaces {
		a.emitReplicaSets(ns)
	}

	if len(msgs) > 0 {
		return fmt.Errorf("failed to delete %d of %d replicasets: %s", len(msgs), len(items), strings.Join(msgs, "; "))
	}
	return nil
}

func (a *App) emitReplicaSets(namespace string) {
	a.mu.RLock()
	h := a.factories[a.activeContext]
	a.mu.RUnlock()
	if h == nil {
		return
	}
	if h.IsForbidden("replicasets") {
		return
	}
	<-h.GetSyncedChan("replicasets")
	if h.IsForbidden("replicasets") {
		return
	}
	lister := h.Factory.Apps().V1().ReplicaSets().Lister()
	allData, err := kubeResources.ListReplicaSets(lister, "")
	if err != nil {
		log.Printf("app: emitReplicaSets: %v", err)
		return
	}
	runtime.EventsEmit(a.ctx, "replicasets:update", allData)
	if namespace != "" {
		nsData, err := kubeResources.ListReplicaSets(lister, namespace)
		if err != nil {
			log.Printf("app: emitReplicaSets ns=%s: %v", namespace, err)
			return
		}
		runtime.EventsEmit(a.ctx, "replicasets:"+namespace+":update", nsData)
	}
}

func (a *App) GetReplicaSetYAML(namespace, name string) (string, error) {
	a.mu.RLock()
	cs := a.clients[a.activeContext]
	a.mu.RUnlock()
	if cs == nil {
		return "", fmt.Errorf("not connected")
	}

	ctx, cancel := context.WithTimeout(context.Background(), apiReadTimeout)
	defer cancel()
	rs, err := cs.AppsV1().ReplicaSets(namespace).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		return "", fmt.Errorf("get ReplicaSet: %w", err)
	}

	yamlBytes, err := sigsyaml.Marshal(rs)
	if err != nil {
		return "", fmt.Errorf("marshal ReplicaSet to YAML: %w", err)
	}

	return string(yamlBytes), nil
}

func (a *App) UpdateReplicaSetYAML(namespace, yamlString string) error {
	a.mu.RLock()
	cs := a.clients[a.activeContext]
	a.mu.RUnlock()
	if cs == nil {
		return fmt.Errorf("not connected")
	}

	var rs appsv1.ReplicaSet
	err := sigsyaml.Unmarshal([]byte(yamlString), &rs)
	if err != nil {
		return fmt.Errorf("unmarshal YAML to ReplicaSet: %w", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), apiMutationTimeout)
	defer cancel()
	_, err = cs.AppsV1().ReplicaSets(namespace).Update(ctx, &rs, metav1.UpdateOptions{})
	if err != nil {
		return fmt.Errorf("update ReplicaSet: %w", err)
	}

	a.emitReplicaSets(namespace)

	return nil
}

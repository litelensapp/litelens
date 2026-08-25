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

func (a *App) ListPersistentVolumeClaims() ([]dto.PersistentVolumeClaim, error) {
	a.mu.RLock()
	h := a.factories[a.activeContext]
	namespaces := a.activeNamespaces
	a.mu.RUnlock()
	if h == nil {
		return []dto.PersistentVolumeClaim{}, nil
	}
	if h.IsForbidden("pvcs") {
		return []dto.PersistentVolumeClaim{}, nil
	}
	<-h.GetSyncedChan("pvcs")
	if h.IsForbidden("pvcs") {
		return nil, nil
	}
	result, err := kubeResources.ListPersistentVolumeClaims(
		h.Factory.Core().V1().PersistentVolumeClaims().Lister(),
		h.Factory.Core().V1().Pods().Lister(),
		namespaces,
	)
	if err != nil {
		log.Printf("app: ListPersistentVolumeClaims: %v", err)
		return []dto.PersistentVolumeClaim{}, nil
	}
	return result, nil
}

func (a *App) GetPersistentVolumeClaimByName(namespace, name string) (*dto.PersistentVolumeClaimDetail, error) {
	a.mu.RLock()
	h := a.factories[a.activeContext]
	a.mu.RUnlock()
	if h == nil {
		return &dto.PersistentVolumeClaimDetail{}, nil
	}
	if h.IsForbidden("pvcs") {
		return &dto.PersistentVolumeClaimDetail{}, nil
	}
	<-h.GetSyncedChan("pvcs")
	if h.IsForbidden("pvcs") {
		return nil, nil
	}
	result, err := kubeResources.GetPersistentVolumeClaimByName(
		h.Factory.Core().V1().PersistentVolumeClaims().Lister(),
		h.Factory.Core().V1().Pods().Lister(),
		namespace,
		name,
	)
	if err != nil {
		log.Printf("app: GetPersistentVolumeClaimByName: %v", err)
		return &dto.PersistentVolumeClaimDetail{}, nil
	}
	return result, nil
}

func (a *App) emitPersistentVolumeClaims() {
	a.mu.RLock()
	h := a.factories[a.activeContext]
	namespaces := a.activeNamespaces
	a.mu.RUnlock()
	if h == nil {
		return
	}
	if h.IsForbidden("pvcs") {
		return
	}
	<-h.GetSyncedChan("pvcs")
	if h.IsForbidden("pvcs") {
		return
	}
	pvcLister := h.Factory.Core().V1().PersistentVolumeClaims().Lister()
	podLister := h.Factory.Core().V1().Pods().Lister()
	data, err := kubeResources.ListPersistentVolumeClaims(pvcLister, podLister, namespaces)
	if err != nil {
		log.Printf("app: emitPersistentVolumeClaims: %v", err)
		return
	}
	runtime.EventsEmit(a.ctx, "pvcs:update", data)
}

// DeletePersistentVolumeClaim deletes a PersistentVolumeClaim from the specified namespace.
func (a *App) DeletePersistentVolumeClaim(namespace, name string) error {
	a.mu.RLock()
	cs := a.clients[a.activeContext]
	a.mu.RUnlock()
	if cs == nil {
		return fmt.Errorf("not connected")
	}

	ctx, cancel := context.WithTimeout(context.Background(), apiMutationTimeout)
	defer cancel()
	err := cs.CoreV1().PersistentVolumeClaims(namespace).Delete(ctx, name, metav1.DeleteOptions{})
	if err != nil && !errors.IsNotFound(err) {
		return fmt.Errorf("delete PersistentVolumeClaim: %w", err)
	}

	a.emitPersistentVolumeClaims()

	return nil
}

// DeletePersistentVolumeClaims deletes multiple PersistentVolumeClaims, handling best-effort deletion across namespaces.
func (a *App) DeletePersistentVolumeClaims(items []dto.PersistentVolumeClaimRef) error {
	a.mu.RLock()
	cs := a.clients[a.activeContext]
	a.mu.RUnlock()
	if cs == nil {
		return fmt.Errorf("not connected")
	}

	var msgs []string

	for _, ref := range items {
		ctx, cancel := context.WithTimeout(context.Background(), apiMutationTimeout)
		err := cs.CoreV1().PersistentVolumeClaims(ref.Namespace).Delete(ctx, ref.Name, metav1.DeleteOptions{})
		cancel()
		if err != nil && !errors.IsNotFound(err) {
			msgs = append(msgs, fmt.Sprintf("%s/%s: %v", ref.Namespace, ref.Name, err))
		}
	}

	a.emitPersistentVolumeClaims()

	if len(msgs) > 0 {
		return fmt.Errorf("failed to delete %d of %d persistentvolumeclaims: %s", len(msgs), len(items), strings.Join(msgs, "; "))
	}
	return nil
}

func (a *App) GetPersistentVolumeClaimYAML(namespace, name string) (string, error) {
	a.mu.RLock()
	cs := a.clients[a.activeContext]
	a.mu.RUnlock()
	if cs == nil {
		return "", fmt.Errorf("not connected")
	}

	ctx, cancel := context.WithTimeout(context.Background(), apiReadTimeout)
	defer cancel()
	pvc, err := cs.CoreV1().PersistentVolumeClaims(namespace).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		return "", fmt.Errorf("get PersistentVolumeClaim: %w", err)
	}

	b, err := sigsyaml.Marshal(pvc)
	if err != nil {
		return "", fmt.Errorf("marshal PersistentVolumeClaim: %w", err)
	}

	return string(b), nil
}

func (a *App) UpdatePersistentVolumeClaimYAML(namespace, yamlString string) error {
	a.mu.RLock()
	cs := a.clients[a.activeContext]
	a.mu.RUnlock()
	if cs == nil {
		return fmt.Errorf("not connected")
	}

	var pvc corev1.PersistentVolumeClaim
	err := sigsyaml.Unmarshal([]byte(yamlString), &pvc)
	if err != nil {
		return fmt.Errorf("unmarshal YAML to PersistentVolumeClaim: %w", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), apiMutationTimeout)
	defer cancel()
	_, err = cs.CoreV1().PersistentVolumeClaims(namespace).Update(ctx, &pvc, metav1.UpdateOptions{})
	if err != nil {
		return fmt.Errorf("update PersistentVolumeClaim: %w", err)
	}

	a.emitPersistentVolumeClaims()

	return nil
}

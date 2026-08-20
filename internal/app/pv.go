package app

import (
	"context"
	"fmt"
	"log"
	"strings"

	"github.com/litelensapp/litelens/packages/core/dto"
	"github.com/litelensapp/litelens/internal/kube/resources"
	"github.com/wailsapp/wails/v2/pkg/runtime"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	sigsyaml "sigs.k8s.io/yaml"
)

func (a *App) ListPersistentVolumes() ([]dto.PersistentVolume, error) {
	a.mu.RLock()
	h := a.factories[a.activeContext]
	a.mu.RUnlock()
	if h == nil {
		return []dto.PersistentVolume{}, nil
	}
	if h.IsForbidden("pvs") {
		return []dto.PersistentVolume{}, nil
	}
	<-h.GetSyncedChan("pvs")
	if h.IsForbidden("pvs") {
		return nil, nil
	}
	result, err := kubeResources.ListPersistentVolumes(
		h.Factory.Core().V1().PersistentVolumes().Lister(),
	)
	if err != nil {
		log.Printf("app: ListPersistentVolumes: %v", err)
		return []dto.PersistentVolume{}, nil
	}
	return result, nil
}

func (a *App) emitPersistentVolumes() {
	a.mu.RLock()
	h := a.factories[a.activeContext]
	a.mu.RUnlock()
	if h == nil {
		return
	}
	if h.IsForbidden("pvs") {
		return
	}
	<-h.GetSyncedChan("pvs")
	if h.IsForbidden("pvs") {
		return
	}
	data, err := kubeResources.ListPersistentVolumes(
		h.Factory.Core().V1().PersistentVolumes().Lister(),
	)
	if err != nil {
		log.Printf("app: emitPersistentVolumes: %v", err)
		return
	}
	runtime.EventsEmit(a.ctx, "pvs:update", data)
}

// DeletePersistentVolume deletes a PersistentVolume.
func (a *App) DeletePersistentVolume(name string) error {
	a.mu.RLock()
	cs := a.clients[a.activeContext]
	a.mu.RUnlock()
	if cs == nil {
		return fmt.Errorf("not connected")
	}

	ctx, cancel := context.WithTimeout(context.Background(), apiMutationTimeout)
	defer cancel()
	err := cs.CoreV1().PersistentVolumes().Delete(ctx, name, metav1.DeleteOptions{})
	if err != nil && !errors.IsNotFound(err) {
		return fmt.Errorf("delete PersistentVolume: %w", err)
	}

	a.emitPersistentVolumes()

	return nil
}

// DeletePersistentVolumes deletes multiple PersistentVolumes.
func (a *App) DeletePersistentVolumes(items []dto.PersistentVolumeRef) error {
	a.mu.RLock()
	cs := a.clients[a.activeContext]
	a.mu.RUnlock()
	if cs == nil {
		return fmt.Errorf("not connected")
	}

	var msgs []string
	for _, ref := range items {
		ctx, cancel := context.WithTimeout(context.Background(), apiMutationTimeout)
		err := cs.CoreV1().PersistentVolumes().Delete(ctx, ref.Name, metav1.DeleteOptions{})
		cancel()
		if err != nil && !errors.IsNotFound(err) {
			msgs = append(msgs, fmt.Sprintf("%s: %v", ref.Name, err))
		}
	}

	a.emitPersistentVolumes()

	if len(msgs) > 0 {
		return fmt.Errorf("failed to delete %d of %d persistentvolumes: %s", len(msgs), len(items), strings.Join(msgs, "; "))
	}
	return nil
}

func (a *App) GetPersistentVolumeYAML(name string) (string, error) {
	a.mu.RLock()
	cs := a.clients[a.activeContext]
	a.mu.RUnlock()
	if cs == nil {
		return "", fmt.Errorf("not connected")
	}

	ctx, cancel := context.WithTimeout(context.Background(), apiReadTimeout)
	defer cancel()
	pv, err := cs.CoreV1().PersistentVolumes().Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		return "", fmt.Errorf("get PersistentVolume: %w", err)
	}

	yamlBytes, err := sigsyaml.Marshal(pv)
	if err != nil {
		return "", fmt.Errorf("marshal PersistentVolume to YAML: %w", err)
	}

	return string(yamlBytes), nil
}

func (a *App) UpdatePersistentVolumeYAML(yamlString string) error {
	a.mu.RLock()
	cs := a.clients[a.activeContext]
	a.mu.RUnlock()
	if cs == nil {
		return fmt.Errorf("not connected")
	}

	var pv corev1.PersistentVolume
	err := sigsyaml.Unmarshal([]byte(yamlString), &pv)
	if err != nil {
		return fmt.Errorf("unmarshal YAML to PersistentVolume: %w", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), apiMutationTimeout)
	defer cancel()
	_, err = cs.CoreV1().PersistentVolumes().Update(ctx, &pv, metav1.UpdateOptions{})
	if err != nil {
		return fmt.Errorf("update PersistentVolume: %w", err)
	}

	a.emitPersistentVolumes()

	return nil
}

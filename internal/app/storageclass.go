package app

import (
	"context"
	"fmt"
	"log"
	"strings"

	"github.com/gknguyen/litelens/internal/dto"
	"github.com/gknguyen/litelens/internal/kube/resources"
	"github.com/wailsapp/wails/v2/pkg/runtime"
	storagev1 "k8s.io/api/storage/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	sigsyaml "sigs.k8s.io/yaml"
)

func (a *App) ListStorageClasses() ([]dto.StorageClass, error) {
	a.mu.RLock()
	h := a.factories[a.activeContext]
	a.mu.RUnlock()
	if h == nil {
		return []dto.StorageClass{}, nil
	}
	if h.IsForbidden("storageclasses") {
		return []dto.StorageClass{}, nil
	}
	<-h.GetSyncedChan("storageclasses")
	if h.IsForbidden("storageclasses") {
		return nil, nil
	}
	result, err := kubeResources.ListStorageClasses(
		h.Factory.Storage().V1().StorageClasses().Lister(),
	)
	if err != nil {
		log.Printf("app: ListStorageClasses: %v", err)
		return []dto.StorageClass{}, nil
	}
	return result, nil
}

func (a *App) GetStorageClassByName(name string) (dto.StorageClass, error) {
	a.mu.RLock()
	h := a.factories[a.activeContext]
	a.mu.RUnlock()
	if h == nil {
		return dto.StorageClass{}, nil
	}
	if h.IsForbidden("storageclasses") {
		return dto.StorageClass{}, nil
	}
	<-h.GetSyncedChan("storageclasses")
	if h.IsForbidden("storageclasses") {
		return dto.StorageClass{}, nil
	}
	result, err := kubeResources.GetStorageClassByName(h.Factory.Storage().V1().StorageClasses().Lister(), name)
	if err != nil {
		log.Printf("app: GetStorageClassByName: %v", err)
		return dto.StorageClass{}, nil
	}
	return result, nil
}

func (a *App) emitStorageClasses() {
	a.mu.RLock()
	h := a.factories[a.activeContext]
	a.mu.RUnlock()
	if h == nil {
		return
	}
	if h.IsForbidden("storageclasses") {
		return
	}
	<-h.GetSyncedChan("storageclasses")
	if h.IsForbidden("storageclasses") {
		return
	}
	data, err := kubeResources.ListStorageClasses(
		h.Factory.Storage().V1().StorageClasses().Lister(),
	)
	if err != nil {
		log.Printf("app: emitStorageClasses: %v", err)
		return
	}
	runtime.EventsEmit(a.ctx, "storageclasses:update", data)
}

// DeleteStorageClass deletes a StorageClass.
func (a *App) DeleteStorageClass(name string) error {
	a.mu.RLock()
	cs := a.clients[a.activeContext]
	a.mu.RUnlock()
	if cs == nil {
		return fmt.Errorf("not connected")
	}

	ctx, cancel := context.WithTimeout(context.Background(), apiMutationTimeout)
	defer cancel()
	err := cs.StorageV1().StorageClasses().Delete(ctx, name, metav1.DeleteOptions{})
	if err != nil && !errors.IsNotFound(err) {
		return fmt.Errorf("delete StorageClass: %w", err)
	}

	a.emitStorageClasses()

	return nil
}

// DeleteStorageClasses deletes multiple StorageClasses.
func (a *App) DeleteStorageClasses(items []dto.StorageClassRef) error {
	a.mu.RLock()
	cs := a.clients[a.activeContext]
	a.mu.RUnlock()
	if cs == nil {
		return fmt.Errorf("not connected")
	}

	var msgs []string
	for _, ref := range items {
		ctx, cancel := context.WithTimeout(context.Background(), apiMutationTimeout)
		err := cs.StorageV1().StorageClasses().Delete(ctx, ref.Name, metav1.DeleteOptions{})
		cancel()
		if err != nil && !errors.IsNotFound(err) {
			msgs = append(msgs, fmt.Sprintf("%s: %v", ref.Name, err))
		}
	}

	a.emitStorageClasses()

	if len(msgs) > 0 {
		return fmt.Errorf("failed to delete %d of %d storageclasses: %s", len(msgs), len(items), strings.Join(msgs, "; "))
	}
	return nil
}

func (a *App) GetStorageClassYAML(name string) (string, error) {
	a.mu.RLock()
	cs := a.clients[a.activeContext]
	a.mu.RUnlock()
	if cs == nil {
		return "", fmt.Errorf("not connected")
	}

	ctx, cancel := context.WithTimeout(context.Background(), apiReadTimeout)
	defer cancel()
	sc, err := cs.StorageV1().StorageClasses().Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		return "", fmt.Errorf("get StorageClass: %w", err)
	}

	yamlBytes, err := sigsyaml.Marshal(sc)
	if err != nil {
		return "", fmt.Errorf("marshal StorageClass to YAML: %w", err)
	}

	return string(yamlBytes), nil
}

func (a *App) UpdateStorageClassYAML(yamlString string) error {
	a.mu.RLock()
	cs := a.clients[a.activeContext]
	a.mu.RUnlock()
	if cs == nil {
		return fmt.Errorf("not connected")
	}

	var sc storagev1.StorageClass
	err := sigsyaml.Unmarshal([]byte(yamlString), &sc)
	if err != nil {
		return fmt.Errorf("unmarshal YAML to StorageClass: %w", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), apiMutationTimeout)
	defer cancel()
	_, err = cs.StorageV1().StorageClasses().Update(ctx, &sc, metav1.UpdateOptions{})
	if err != nil {
		return fmt.Errorf("update StorageClass: %w", err)
	}

	a.emitStorageClasses()

	return nil
}

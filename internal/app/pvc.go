package app

import (
	"context"
	"fmt"
	"log"

	kubeResources "github.com/litelensapp/litelens/internal/kube/resources"
	"github.com/litelensapp/litelens/packages/core/kube/dto"
	"github.com/wailsapp/wails/v2/pkg/runtime"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	sigsyaml "sigs.k8s.io/yaml"
)

func (a *App) ListPersistentVolumeClaims() ([]dto.PersistentVolumeClaim, error) {
	h, namespaces := a.activeFactoryAndNamespaces()
	if !waitForResourceSync(h, "pvcs") {
		return []dto.PersistentVolumeClaim{}, nil
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
	h := a.activeFactory()
	if !waitForResourceSync(h, "pvcs") {
		return &dto.PersistentVolumeClaimDetail{}, nil
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
	h, namespaces := a.activeFactoryAndNamespaces()
	if !waitForResourceSync(h, "pvcs") {
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
	cs, err := a.activeClientset()
	if err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(context.Background(), apiMutationTimeout)
	defer cancel()
	err = cs.CoreV1().PersistentVolumeClaims(namespace).Delete(ctx, name, metav1.DeleteOptions{})
	if err != nil && !errors.IsNotFound(err) {
		return fmt.Errorf("delete PersistentVolumeClaim: %w", err)
	}

	a.emitPersistentVolumeClaims()

	return nil
}

// DeletePersistentVolumeClaims deletes multiple PersistentVolumeClaims, handling best-effort deletion across namespaces.
func (a *App) DeletePersistentVolumeClaims(items []dto.PersistentVolumeClaimRef) error {
	cs, err := a.activeClientset()
	if err != nil {
		return err
	}

	err = deleteRefsBestEffort(items,
		func(r dto.PersistentVolumeClaimRef) string { return r.Namespace },
		func(r dto.PersistentVolumeClaimRef) string { return r.Name },
		"persistentvolumeclaims",
		func(ctx context.Context, namespace, name string) error {
			return cs.CoreV1().PersistentVolumeClaims(namespace).Delete(ctx, name, metav1.DeleteOptions{})
		},
	)

	a.emitPersistentVolumeClaims()

	return err
}

func (a *App) GetPersistentVolumeClaimYAML(namespace, name string) (string, error) {
	cs, err := a.activeClientset()
	if err != nil {
		return "", err
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
	cs, err := a.activeClientset()
	if err != nil {
		return err
	}

	var pvc corev1.PersistentVolumeClaim
	err = sigsyaml.Unmarshal([]byte(yamlString), &pvc)
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

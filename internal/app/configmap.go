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

func (a *App) ListConfigMaps(namespaces []string) ([]dto.ConfigMap, error) {
	a.mu.RLock()
	h := a.factories[a.activeContext]
	a.mu.RUnlock()
	if h == nil {
		return []dto.ConfigMap{}, nil
	}
	if h.IsForbidden("configmaps") {
		return []dto.ConfigMap{}, nil
	}
	<-h.GetSyncedChan("configmaps")
	if h.IsForbidden("configmaps") {
		return nil, nil
	}
	result, err := kubeResources.ListConfigMaps(h.Factory.Core().V1().ConfigMaps().Lister(), namespaces)
	if err != nil {
		log.Printf("app: ListConfigMaps: %v", err)
		return []dto.ConfigMap{}, nil
	}
	return result, nil
}

func (a *App) GetConfigMapByName(namespace, name string) (dto.ConfigMap, error) {
	a.mu.RLock()
	h := a.factories[a.activeContext]
	a.mu.RUnlock()
	if h == nil {
		return dto.ConfigMap{}, nil
	}
	if h.IsForbidden("configmaps") {
		return dto.ConfigMap{}, nil
	}
	<-h.GetSyncedChan("configmaps")
	if h.IsForbidden("configmaps") {
		return dto.ConfigMap{}, nil
	}
	result, err := kubeResources.GetConfigMapByName(h.Factory.Core().V1().ConfigMaps().Lister(), namespace, name)
	if err != nil {
		log.Printf("app: GetConfigMapByName: %v", err)
		return dto.ConfigMap{}, nil
	}
	return result, nil
}

func (a *App) UpdateConfigMap(namespace, name string, data map[string]string) error {
	a.mu.RLock()
	cs := a.clients[a.activeContext]
	a.mu.RUnlock()
	if cs == nil {
		return fmt.Errorf("not connected")
	}
	ctx, cancel := context.WithTimeout(context.Background(), apiMutationTimeout)
	defer cancel()
	cm, err := cs.CoreV1().ConfigMaps(namespace).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		return err
	}
	cm.Data = data
	_, err = cs.CoreV1().ConfigMaps(namespace).Update(ctx, cm, metav1.UpdateOptions{})
	return err
}

func (a *App) DeleteConfigMap(namespace, name string) error {
	a.mu.RLock()
	cs := a.clients[a.activeContext]
	a.mu.RUnlock()
	if cs == nil {
		return fmt.Errorf("not connected")
	}

	ctx, cancel := context.WithTimeout(context.Background(), apiMutationTimeout)
	defer cancel()
	err := cs.CoreV1().ConfigMaps(namespace).Delete(ctx, name, metav1.DeleteOptions{})
	if err != nil && !errors.IsNotFound(err) {
		return fmt.Errorf("delete ConfigMap: %w", err)
	}

	a.emitConfigMaps([]string{namespace})

	return nil
}

// DeleteConfigMaps deletes multiple ConfigMaps, handling best-effort deletion across namespaces.
func (a *App) DeleteConfigMaps(items []dto.ConfigMapRef) error {
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
		err := cs.CoreV1().ConfigMaps(ref.Namespace).Delete(ctx, ref.Name, metav1.DeleteOptions{})
		cancel()
		if err != nil && !errors.IsNotFound(err) {
			msgs = append(msgs, fmt.Sprintf("%s/%s: %v", ref.Namespace, ref.Name, err))
		}
		namespaces[ref.Namespace] = true
	}

	// Emit updates for each unique namespace touched
	touchedNamespaces := make([]string, 0, len(namespaces))
	for ns := range namespaces {
		touchedNamespaces = append(touchedNamespaces, ns)
	}
	a.emitConfigMaps(touchedNamespaces)

	if len(msgs) > 0 {
		return fmt.Errorf("failed to delete %d of %d configmaps: %s", len(msgs), len(items), strings.Join(msgs, "; "))
	}
	return nil
}

func (a *App) emitConfigMaps(namespaces []string) {
	a.mu.RLock()
	h := a.factories[a.activeContext]
	a.mu.RUnlock()
	if h == nil {
		return
	}
	if h.IsForbidden("configmaps") {
		return
	}
	<-h.GetSyncedChan("configmaps")
	if h.IsForbidden("configmaps") {
		return
	}
	lister := h.Factory.Core().V1().ConfigMaps().Lister()
	allData, err := kubeResources.ListConfigMaps(lister, nil)
	if err != nil {
		log.Printf("app: emitConfigMaps: %v", err)
		return
	}
	runtime.EventsEmit(a.ctx, "configmaps:update", allData)
	for _, ns := range namespaces {
		// Filter already-fetched cluster-wide data instead of re-listing
		nsData := make([]dto.ConfigMap, 0)
		for _, item := range allData {
			if item.Namespace == ns {
				nsData = append(nsData, item)
			}
		}
		runtime.EventsEmit(a.ctx, "configmaps:"+ns+":update", nsData)
	}
}

func (a *App) GetConfigMapYAML(namespace, name string) (string, error) {
	a.mu.RLock()
	cs := a.clients[a.activeContext]
	a.mu.RUnlock()
	if cs == nil {
		return "", fmt.Errorf("not connected")
	}

	ctx, cancel := context.WithTimeout(context.Background(), apiReadTimeout)
	defer cancel()
	cm, err := cs.CoreV1().ConfigMaps(namespace).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		return "", fmt.Errorf("get ConfigMap: %w", err)
	}

	yamlBytes, err := sigsyaml.Marshal(cm)
	if err != nil {
		return "", fmt.Errorf("marshal ConfigMap to YAML: %w", err)
	}

	return string(yamlBytes), nil
}

func (a *App) UpdateConfigMapYAML(namespace, yamlString string) error {
	a.mu.RLock()
	cs := a.clients[a.activeContext]
	a.mu.RUnlock()
	if cs == nil {
		return fmt.Errorf("not connected")
	}

	var cm corev1.ConfigMap
	err := sigsyaml.Unmarshal([]byte(yamlString), &cm)
	if err != nil {
		return fmt.Errorf("unmarshal YAML to ConfigMap: %w", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), apiMutationTimeout)
	defer cancel()
	_, err = cs.CoreV1().ConfigMaps(namespace).Update(ctx, &cm, metav1.UpdateOptions{})
	if err != nil {
		return fmt.Errorf("update ConfigMap: %w", err)
	}

	a.emitConfigMaps([]string{namespace})

	return nil
}

package app

import (
	"context"
	"fmt"
	"log"
	"strings"

	"github.com/litelensapp/litelens/internal/kube/resources"
	"github.com/litelensapp/litelens/packages/core/kube/dto"
	"github.com/wailsapp/wails/v2/pkg/runtime"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	sigsyaml "sigs.k8s.io/yaml"
)

func (a *App) ListConfigMaps() ([]dto.ConfigMap, error) {
	a.mu.RLock()
	h := a.factories[a.activeContext]
	namespaces := a.activeNamespaces
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

	a.emitConfigMaps()

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

	for _, ref := range items {
		ctx, cancel := context.WithTimeout(context.Background(), apiMutationTimeout)
		err := cs.CoreV1().ConfigMaps(ref.Namespace).Delete(ctx, ref.Name, metav1.DeleteOptions{})
		cancel()
		if err != nil && !errors.IsNotFound(err) {
			msgs = append(msgs, fmt.Sprintf("%s/%s: %v", ref.Namespace, ref.Name, err))
		}
	}

	a.emitConfigMaps()

	if len(msgs) > 0 {
		return fmt.Errorf("failed to delete %d of %d configmaps: %s", len(msgs), len(items), strings.Join(msgs, "; "))
	}
	return nil
}

func (a *App) emitConfigMaps() {
	a.mu.RLock()
	h := a.factories[a.activeContext]
	namespaces := a.activeNamespaces
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
	data, err := kubeResources.ListConfigMaps(lister, namespaces)
	if err != nil {
		log.Printf("app: emitConfigMaps: %v", err)
		return
	}
	runtime.EventsEmit(a.ctx, "configmaps:update", data)
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

	a.emitConfigMaps()

	return nil
}

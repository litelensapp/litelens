package app

import (
	"context"
	"fmt"
	"log"
	"strings"

	"github.com/litelensapp/litelens/packages/core/kube/dto"
	kubeResources "github.com/litelensapp/litelens/internal/kube/resources"
	"github.com/wailsapp/wails/v2/pkg/runtime"
	admissionregistrationv1 "k8s.io/api/admissionregistration/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	sigsyaml "sigs.k8s.io/yaml"
)

func (a *App) ListValidatingWebhookConfigs() ([]dto.ValidatingWebhookConfig, error) {
	a.mu.RLock()
	h := a.factories[a.activeContext]
	a.mu.RUnlock()
	if h == nil {
		return []dto.ValidatingWebhookConfig{}, nil
	}
	if h.IsForbidden("validatingwebhookconfigs") {
		return []dto.ValidatingWebhookConfig{}, nil
	}
	<-h.GetSyncedChan("validatingwebhookconfigs")
	if h.IsForbidden("validatingwebhookconfigs") {
		return nil, nil
	}
	result, err := kubeResources.ListValidatingWebhookConfigs(h.Factory.Admissionregistration().V1().ValidatingWebhookConfigurations().Lister())
	if err != nil {
		log.Printf("app: ListValidatingWebhookConfigs: %v", err)
		return []dto.ValidatingWebhookConfig{}, nil
	}
	return result, nil
}

func (a *App) GetValidatingWebhookConfigByName(name string) (*dto.ValidatingWebhookConfigDetail, error) {
	a.mu.RLock()
	h := a.factories[a.activeContext]
	a.mu.RUnlock()
	if h == nil {
		return nil, nil
	}
	if h.IsForbidden("validatingwebhookconfigs") {
		return nil, nil
	}
	<-h.GetSyncedChan("validatingwebhookconfigs")
	if h.IsForbidden("validatingwebhookconfigs") {
		return nil, nil
	}
	result, err := kubeResources.GetValidatingWebhookConfigByName(
		h.Factory.Admissionregistration().V1().ValidatingWebhookConfigurations().Lister(),
		name,
	)
	if err != nil {
		log.Printf("app: GetValidatingWebhookConfigByName: %v", err)
		return nil, err
	}
	return result, nil
}

func (a *App) DeleteValidatingWebhookConfig(name string) error {
	a.mu.RLock()
	cs := a.clients[a.activeContext]
	a.mu.RUnlock()
	if cs == nil {
		return fmt.Errorf("not connected")
	}

	ctx, cancel := context.WithTimeout(context.Background(), apiMutationTimeout)
	defer cancel()
	err := cs.AdmissionregistrationV1().ValidatingWebhookConfigurations().Delete(ctx, name, metav1.DeleteOptions{})
	if err != nil && !errors.IsNotFound(err) {
		return fmt.Errorf("delete ValidatingWebhookConfig: %w", err)
	}

	a.emitValidatingWebhookConfigs()

	return nil
}

// DeleteValidatingWebhookConfigs deletes multiple ValidatingWebhookConfigs, handling best-effort deletion.
func (a *App) DeleteValidatingWebhookConfigs(names []string) error {
	a.mu.RLock()
	cs := a.clients[a.activeContext]
	a.mu.RUnlock()
	if cs == nil {
		return fmt.Errorf("not connected")
	}

	var msgs []string
	for _, name := range names {
		ctx, cancel := context.WithTimeout(context.Background(), apiMutationTimeout)
		err := cs.AdmissionregistrationV1().ValidatingWebhookConfigurations().Delete(ctx, name, metav1.DeleteOptions{})
		cancel()
		if err != nil && !errors.IsNotFound(err) {
			msgs = append(msgs, fmt.Sprintf("%s: %v", name, err))
		}
	}

	a.emitValidatingWebhookConfigs()

	if len(msgs) > 0 {
		return fmt.Errorf("failed to delete %d of %d validatingwebhookconfigs: %s", len(msgs), len(names), strings.Join(msgs, "; "))
	}
	return nil
}

func (a *App) emitValidatingWebhookConfigs() {
	a.mu.RLock()
	h := a.factories[a.activeContext]
	a.mu.RUnlock()
	if h == nil {
		return
	}
	if h.IsForbidden("validatingwebhookconfigs") {
		return
	}
	<-h.GetSyncedChan("validatingwebhookconfigs")
	if h.IsForbidden("validatingwebhookconfigs") {
		return
	}
	data, err := kubeResources.ListValidatingWebhookConfigs(h.Factory.Admissionregistration().V1().ValidatingWebhookConfigurations().Lister())
	if err != nil {
		log.Printf("app: emitValidatingWebhookConfigs: %v", err)
		return
	}
	runtime.EventsEmit(a.ctx, "validatingwebhookconfigs:update", data)
}

func (a *App) GetValidatingWebhookConfigYAML(name string) (string, error) {
	a.mu.RLock()
	cs := a.clients[a.activeContext]
	a.mu.RUnlock()
	if cs == nil {
		return "", fmt.Errorf("not connected")
	}

	ctx, cancel := context.WithTimeout(context.Background(), apiReadTimeout)
	defer cancel()
	vwc, err := cs.AdmissionregistrationV1().ValidatingWebhookConfigurations().Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		return "", fmt.Errorf("get ValidatingWebhookConfig: %w", err)
	}

	yamlBytes, err := sigsyaml.Marshal(vwc)
	if err != nil {
		return "", fmt.Errorf("marshal ValidatingWebhookConfig to YAML: %w", err)
	}

	return string(yamlBytes), nil
}

func (a *App) UpdateValidatingWebhookConfigYAML(yamlString string) error {
	a.mu.RLock()
	cs := a.clients[a.activeContext]
	a.mu.RUnlock()
	if cs == nil {
		return fmt.Errorf("not connected")
	}

	var vwc admissionregistrationv1.ValidatingWebhookConfiguration
	err := sigsyaml.Unmarshal([]byte(yamlString), &vwc)
	if err != nil {
		return fmt.Errorf("unmarshal YAML to ValidatingWebhookConfig: %w", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), apiMutationTimeout)
	defer cancel()
	_, err = cs.AdmissionregistrationV1().ValidatingWebhookConfigurations().Update(ctx, &vwc, metav1.UpdateOptions{})
	if err != nil {
		return fmt.Errorf("update ValidatingWebhookConfig: %w", err)
	}

	a.emitValidatingWebhookConfigs()

	return nil
}

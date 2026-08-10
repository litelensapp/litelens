package app

import (
	"context"
	"encoding/base64"
	"fmt"
	"log"
	"strings"

	"github.com/litelensapp/litelens/internal/dto"
	"github.com/litelensapp/litelens/internal/kube/resources"
	"github.com/wailsapp/wails/v2/pkg/runtime"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	sigsyaml "sigs.k8s.io/yaml"
)

func (a *App) ListSecrets(namespace string) ([]dto.Secret, error) {
	a.mu.RLock()
	h := a.factories[a.activeContext]
	a.mu.RUnlock()
	if h == nil {
		return []dto.Secret{}, nil
	}
	if h.IsForbidden("secrets") {
		return []dto.Secret{}, nil
	}
	<-h.GetSyncedChan("secrets")
	if h.IsForbidden("secrets") {
		return nil, nil
	}
	result, err := kubeResources.ListSecrets(h.Factory.Core().V1().Secrets().Lister(), namespace)
	if err != nil {
		log.Printf("app: ListSecrets: %v", err)
		return []dto.Secret{}, nil
	}
	return result, nil
}

func (a *App) GetSecretByName(namespace, name string) (*dto.SecretDetail, error) {
	a.mu.RLock()
	h := a.factories[a.activeContext]
	a.mu.RUnlock()
	if h == nil {
		return &dto.SecretDetail{}, nil
	}
	if h.IsForbidden("secrets") {
		return &dto.SecretDetail{}, nil
	}
	<-h.GetSyncedChan("secrets")
	if h.IsForbidden("secrets") {
		return nil, nil
	}
	result, err := kubeResources.GetSecretByName(
		h.Factory.Core().V1().Secrets().Lister(),
		namespace,
		name,
	)
	if err != nil {
		log.Printf("app: GetSecretByName: %v", err)
		return &dto.SecretDetail{}, nil
	}
	return result, nil
}

func (a *App) UpdateSecret(namespace, name string, data map[string]string) error {
	a.mu.RLock()
	cs := a.clients[a.activeContext]
	a.mu.RUnlock()
	if cs == nil {
		return fmt.Errorf("not connected")
	}
	ctx, cancel := context.WithTimeout(context.Background(), apiMutationTimeout)
	defer cancel()
	secret, err := cs.CoreV1().Secrets(namespace).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		return err
	}
	newData := make(map[string][]byte, len(data))
	for k, v := range data {
		decoded, err := base64.StdEncoding.DecodeString(v)
		if err != nil {
			return fmt.Errorf("invalid base64 for key %q: %w", k, err)
		}
		newData[k] = decoded
	}
	secret.Data = newData
	_, err = cs.CoreV1().Secrets(namespace).Update(ctx, secret, metav1.UpdateOptions{})
	return err
}

func (a *App) DeleteSecret(namespace, name string) error {
	a.mu.RLock()
	cs := a.clients[a.activeContext]
	a.mu.RUnlock()
	if cs == nil {
		return fmt.Errorf("not connected")
	}

	ctx, cancel := context.WithTimeout(context.Background(), apiMutationTimeout)
	defer cancel()
	err := cs.CoreV1().Secrets(namespace).Delete(ctx, name, metav1.DeleteOptions{})
	if err != nil && !errors.IsNotFound(err) {
		return fmt.Errorf("delete Secret: %w", err)
	}

	a.emitSecrets(namespace)

	return nil
}

// DeleteSecrets deletes multiple Secrets, handling best-effort deletion across namespaces.
func (a *App) DeleteSecrets(items []dto.SecretRef) error {
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
		err := cs.CoreV1().Secrets(ref.Namespace).Delete(ctx, ref.Name, metav1.DeleteOptions{})
		cancel()
		if err != nil && !errors.IsNotFound(err) {
			msgs = append(msgs, fmt.Sprintf("%s/%s: %v", ref.Namespace, ref.Name, err))
		}
		namespaces[ref.Namespace] = true
	}

	// Emit updates for each unique namespace touched
	for ns := range namespaces {
		a.emitSecrets(ns)
	}

	if len(msgs) > 0 {
		return fmt.Errorf("failed to delete %d of %d secrets: %s", len(msgs), len(items), strings.Join(msgs, "; "))
	}
	return nil
}

func (a *App) emitSecrets(namespace string) {
	a.mu.RLock()
	h := a.factories[a.activeContext]
	a.mu.RUnlock()
	if h == nil {
		return
	}
	if h.IsForbidden("secrets") {
		return
	}
	<-h.GetSyncedChan("secrets")
	if h.IsForbidden("secrets") {
		return
	}
	lister := h.Factory.Core().V1().Secrets().Lister()
	allData, err := kubeResources.ListSecrets(lister, "")
	if err != nil {
		log.Printf("app: emitSecrets: %v", err)
		return
	}
	runtime.EventsEmit(a.ctx, "secrets:update", allData)
	if namespace != "" {
		nsData, err := kubeResources.ListSecrets(lister, namespace)
		if err != nil {
			log.Printf("app: emitSecrets ns=%s: %v", namespace, err)
			return
		}
		runtime.EventsEmit(a.ctx, "secrets:"+namespace+":update", nsData)
	}
}

func (a *App) GetSecretYAML(namespace, name string) (string, error) {
	a.mu.RLock()
	cs := a.clients[a.activeContext]
	a.mu.RUnlock()
	if cs == nil {
		return "", fmt.Errorf("not connected")
	}

	ctx, cancel := context.WithTimeout(context.Background(), apiReadTimeout)
	defer cancel()
	secret, err := cs.CoreV1().Secrets(namespace).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		return "", fmt.Errorf("get Secret: %w", err)
	}

	yamlBytes, err := sigsyaml.Marshal(secret)
	if err != nil {
		return "", fmt.Errorf("marshal Secret to YAML: %w", err)
	}

	return string(yamlBytes), nil
}

func (a *App) UpdateSecretYAML(namespace, yamlString string) error {
	a.mu.RLock()
	cs := a.clients[a.activeContext]
	a.mu.RUnlock()
	if cs == nil {
		return fmt.Errorf("not connected")
	}

	var secret corev1.Secret
	err := sigsyaml.Unmarshal([]byte(yamlString), &secret)
	if err != nil {
		return fmt.Errorf("unmarshal YAML to Secret: %w", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), apiMutationTimeout)
	defer cancel()
	_, err = cs.CoreV1().Secrets(namespace).Update(ctx, &secret, metav1.UpdateOptions{})
	if err != nil {
		return fmt.Errorf("update Secret: %w", err)
	}

	a.emitSecrets(namespace)

	return nil
}

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

func (a *App) GetServiceAccountByName(namespace, name string) (dto.ServiceAccount, error) {
	h := a.activeFactory()
	if !waitForResourceSync(h, "serviceaccounts") {
		return dto.ServiceAccount{}, nil
	}
	result, err := kubeResources.GetServiceAccountByName(
		h.Factory.Core().V1().ServiceAccounts().Lister(),
		namespace,
		name,
	)
	if err != nil {
		log.Printf("app: GetServiceAccountByName: %v", err)
		return dto.ServiceAccount{}, nil
	}
	return result, nil
}

func (a *App) ListServiceAccounts() ([]dto.ServiceAccount, error) {
	h, namespaces := a.activeFactoryAndNamespaces()
	if !waitForResourceSync(h, "serviceaccounts") {
		return []dto.ServiceAccount{}, nil
	}
	result, err := kubeResources.ListServiceAccounts(
		h.Factory.Core().V1().ServiceAccounts().Lister(),
		namespaces,
	)
	if err != nil {
		log.Printf("app: ListServiceAccounts: %v", err)
		return []dto.ServiceAccount{}, nil
	}
	return result, nil
}

func (a *App) DeleteServiceAccount(namespace, name string) error {
	cs, err := a.activeClientset()
	if err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(context.Background(), apiMutationTimeout)
	defer cancel()
	err = cs.CoreV1().ServiceAccounts(namespace).Delete(ctx, name, metav1.DeleteOptions{})
	if err != nil && !errors.IsNotFound(err) {
		return fmt.Errorf("delete ServiceAccount: %w", err)
	}

	a.emitServiceAccounts()

	return nil
}

// DeleteServiceAccounts deletes multiple ServiceAccounts, handling best-effort deletion across namespaces.
func (a *App) DeleteServiceAccounts(items []dto.ServiceAccountRef) error {
	cs, err := a.activeClientset()
	if err != nil {
		return err
	}

	err = deleteRefsBestEffort(items,
		func(r dto.ServiceAccountRef) string { return r.Namespace },
		func(r dto.ServiceAccountRef) string { return r.Name },
		"serviceaccounts",
		func(ctx context.Context, namespace, name string) error {
			return cs.CoreV1().ServiceAccounts(namespace).Delete(ctx, name, metav1.DeleteOptions{})
		},
	)

	a.emitServiceAccounts()

	return err
}

func (a *App) emitServiceAccounts() {
	h, namespaces := a.activeFactoryAndNamespaces()
	if !waitForResourceSync(h, "serviceaccounts") {
		return
	}
	lister := h.Factory.Core().V1().ServiceAccounts().Lister()
	data, err := kubeResources.ListServiceAccounts(lister, namespaces)
	if err != nil {
		log.Printf("app: emitServiceAccounts: %v", err)
		return
	}
	runtime.EventsEmit(a.ctx, "serviceaccounts:update", data)
}

func (a *App) GetServiceAccountYAML(namespace, name string) (string, error) {
	cs, err := a.activeClientset()
	if err != nil {
		return "", err
	}

	ctx, cancel := context.WithTimeout(context.Background(), apiReadTimeout)
	defer cancel()
	sa, err := cs.CoreV1().ServiceAccounts(namespace).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		return "", fmt.Errorf("get ServiceAccount: %w", err)
	}

	yamlBytes, err := sigsyaml.Marshal(sa)
	if err != nil {
		return "", fmt.Errorf("marshal ServiceAccount to YAML: %w", err)
	}

	return string(yamlBytes), nil
}

func (a *App) UpdateServiceAccountYAML(namespace, yamlString string) error {
	cs, err := a.activeClientset()
	if err != nil {
		return err
	}

	var sa corev1.ServiceAccount
	err = sigsyaml.Unmarshal([]byte(yamlString), &sa)
	if err != nil {
		return fmt.Errorf("unmarshal YAML to ServiceAccount: %w", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), apiMutationTimeout)
	defer cancel()
	_, err = cs.CoreV1().ServiceAccounts(namespace).Update(ctx, &sa, metav1.UpdateOptions{})
	if err != nil {
		return fmt.Errorf("update ServiceAccount: %w", err)
	}

	a.emitServiceAccounts()

	return nil
}

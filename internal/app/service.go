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

func (a *App) GetServiceByName(namespace, name string) (dto.Service, error) {
	h := a.activeFactory()
	if h == nil {
		return dto.Service{}, nil
	}
	result, err := kubeResources.GetServiceByName(h.Factory.Core().V1().Services().Lister(), namespace, name)
	if err != nil {
		log.Printf("app: GetServiceByName: %v", err)
		return dto.Service{}, err
	}
	return result, nil
}

func (a *App) ListServices() ([]dto.Service, error) {
	h, namespaces := a.activeFactoryAndNamespaces()
	if !waitForResourceSync(h, "services") {
		return []dto.Service{}, nil
	}
	result, err := kubeResources.ListServices(h.Factory.Core().V1().Services().Lister(), namespaces)
	if err != nil {
		log.Printf("app: ListServices: %v", err)
		return []dto.Service{}, nil
	}
	return result, nil
}

// DeleteService deletes a Service from the specified namespace.
func (a *App) DeleteService(namespace, name string) error {
	cs, err := a.activeClientset()
	if err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(context.Background(), apiMutationTimeout)
	defer cancel()
	err = cs.CoreV1().Services(namespace).Delete(ctx, name, metav1.DeleteOptions{})
	if err != nil && !errors.IsNotFound(err) {
		return fmt.Errorf("delete Service: %w", err)
	}

	// Emit update event after successful delete
	a.emitServices()

	return nil
}

// DeleteServices deletes multiple Services, handling best-effort deletion across namespaces.
func (a *App) DeleteServices(items []dto.ServiceRef) error {
	cs, err := a.activeClientset()
	if err != nil {
		return err
	}

	err = deleteRefsBestEffort(items,
		func(r dto.ServiceRef) string { return r.Namespace },
		func(r dto.ServiceRef) string { return r.Name },
		"services",
		func(ctx context.Context, namespace, name string) error {
			return cs.CoreV1().Services(namespace).Delete(ctx, name, metav1.DeleteOptions{})
		},
	)

	a.emitServices()

	return err
}

func (a *App) emitServices() {
	h, namespaces := a.activeFactoryAndNamespaces()
	if !waitForResourceSync(h, "services") {
		return
	}
	lister := h.Factory.Core().V1().Services().Lister()
	data, err := kubeResources.ListServices(lister, namespaces)
	if err != nil {
		log.Printf("app: emitServices: %v", err)
		return
	}
	runtime.EventsEmit(a.ctx, "services:update", data)
}

func (a *App) GetServiceYAML(namespace, name string) (string, error) {
	cs, err := a.activeClientset()
	if err != nil {
		return "", err
	}

	ctx, cancel := context.WithTimeout(context.Background(), apiReadTimeout)
	defer cancel()
	svc, err := cs.CoreV1().Services(namespace).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		return "", fmt.Errorf("get Service: %w", err)
	}

	yamlBytes, err := sigsyaml.Marshal(svc)
	if err != nil {
		return "", fmt.Errorf("marshal Service to YAML: %w", err)
	}

	return string(yamlBytes), nil
}

func (a *App) UpdateServiceYAML(namespace, yamlString string) error {
	cs, err := a.activeClientset()
	if err != nil {
		return err
	}

	var svc corev1.Service
	err = sigsyaml.Unmarshal([]byte(yamlString), &svc)
	if err != nil {
		return fmt.Errorf("unmarshal YAML to Service: %w", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), apiMutationTimeout)
	defer cancel()
	_, err = cs.CoreV1().Services(namespace).Update(ctx, &svc, metav1.UpdateOptions{})
	if err != nil {
		return fmt.Errorf("update Service: %w", err)
	}

	a.emitServices()

	return nil
}

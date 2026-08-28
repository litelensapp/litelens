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
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	sigsyaml "sigs.k8s.io/yaml"
)

func (a *App) ListResourceQuotas() ([]dto.ResourceQuota, error) {
	h, namespaces := a.activeFactoryAndNamespaces()
	if !waitForResourceSync(h, "resourcequotas") {
		return []dto.ResourceQuota{}, nil
	}
	result, err := kubeResources.ListResourceQuotas(h.Factory.Core().V1().ResourceQuotas().Lister(), namespaces)
	if err != nil {
		log.Printf("app: ListResourceQuotas: %v", err)
		return []dto.ResourceQuota{}, nil
	}
	return result, nil
}

func (a *App) GetResourceQuotaByName(namespace, name string) (dto.ResourceQuotaDetail, error) {
	h := a.activeFactory()
	if !waitForResourceSync(h, "resourcequotas") {
		return dto.ResourceQuotaDetail{}, nil
	}
	result, err := kubeResources.GetResourceQuotaByName(h.Factory.Core().V1().ResourceQuotas().Lister(), namespace, name)
	if err != nil {
		log.Printf("app: GetResourceQuotaByName: %v", err)
		return dto.ResourceQuotaDetail{}, nil
	}
	return result, nil
}

func (a *App) CreateResourceQuota(namespace, name string, hard map[string]string) error {
	cs, err := a.activeClientset()
	if err != nil {
		return err
	}

	resourceList := make(corev1.ResourceList)
	for k, v := range hard {
		qty, err := resource.ParseQuantity(v)
		if err != nil {
			return fmt.Errorf("invalid quantity for %q: %w", k, err)
		}
		resourceList[corev1.ResourceName(k)] = qty
	}

	rq := &corev1.ResourceQuota{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		Spec: corev1.ResourceQuotaSpec{
			Hard: resourceList,
		},
	}

	ctx, cancel := context.WithTimeout(context.Background(), apiMutationTimeout)
	defer cancel()
	_, err = cs.CoreV1().ResourceQuotas(namespace).Create(ctx, rq, metav1.CreateOptions{})
	return err
}

// DeleteResourceQuota deletes a ResourceQuota from the specified namespace.
func (a *App) DeleteResourceQuota(namespace, name string) error {
	cs, err := a.activeClientset()
	if err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(context.Background(), apiMutationTimeout)
	defer cancel()
	err = cs.CoreV1().ResourceQuotas(namespace).Delete(ctx, name, metav1.DeleteOptions{})
	if err != nil && !errors.IsNotFound(err) {
		return fmt.Errorf("delete ResourceQuota: %w", err)
	}

	// Emit update event after successful delete
	a.emitResourceQuotas()

	return nil
}

// DeleteResourceQuotas deletes multiple ResourceQuotas, handling best-effort deletion across namespaces.
func (a *App) DeleteResourceQuotas(items []dto.ResourceQuotaRef) error {
	cs, err := a.activeClientset()
	if err != nil {
		return err
	}

	err = deleteRefsBestEffort(items,
		func(r dto.ResourceQuotaRef) string { return r.Namespace },
		func(r dto.ResourceQuotaRef) string { return r.Name },
		"resourcequotas",
		func(ctx context.Context, namespace, name string) error {
			return cs.CoreV1().ResourceQuotas(namespace).Delete(ctx, name, metav1.DeleteOptions{})
		},
	)

	a.emitResourceQuotas()

	return err
}

func (a *App) emitResourceQuotas() {
	h, namespaces := a.activeFactoryAndNamespaces()
	if !waitForResourceSync(h, "resourcequotas") {
		return
	}
	lister := h.Factory.Core().V1().ResourceQuotas().Lister()
	data, err := kubeResources.ListResourceQuotas(lister, namespaces)
	if err != nil {
		log.Printf("app: emitResourceQuotas: %v", err)
		return
	}
	runtime.EventsEmit(a.ctx, "resourcequotas:update", data)
}

func (a *App) GetResourceQuotaYAML(namespace, name string) (string, error) {
	cs, err := a.activeClientset()
	if err != nil {
		return "", err
	}

	ctx, cancel := context.WithTimeout(context.Background(), apiReadTimeout)
	defer cancel()
	rq, err := cs.CoreV1().ResourceQuotas(namespace).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		return "", fmt.Errorf("get ResourceQuota: %w", err)
	}

	b, err := sigsyaml.Marshal(rq)
	if err != nil {
		return "", fmt.Errorf("marshal ResourceQuota to YAML: %w", err)
	}

	return string(b), nil
}

func (a *App) UpdateResourceQuotaYAML(namespace, yamlString string) error {
	cs, err := a.activeClientset()
	if err != nil {
		return err
	}

	var rq corev1.ResourceQuota
	err = sigsyaml.Unmarshal([]byte(yamlString), &rq)
	if err != nil {
		return fmt.Errorf("unmarshal YAML to ResourceQuota: %w", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), apiMutationTimeout)
	defer cancel()
	_, err = cs.CoreV1().ResourceQuotas(namespace).Update(ctx, &rq, metav1.UpdateOptions{})
	if err != nil {
		return fmt.Errorf("update ResourceQuota: %w", err)
	}

	a.emitResourceQuotas()

	return nil
}

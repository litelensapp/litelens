package app

import (
	"context"
	"fmt"
	"log"

	kubeResources "github.com/litelensapp/litelens/internal/kube/resources"
	"github.com/litelensapp/litelens/packages/core/kube/dto"
	"github.com/wailsapp/wails/v2/pkg/runtime"
	autoscalingv2 "k8s.io/api/autoscaling/v2"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	sigsyaml "sigs.k8s.io/yaml"
)

func (a *App) GetHPAByName(namespace, name string) (dto.HPADetail, error) {
	h := a.activeFactory()
	if !waitForResourceSync(h, "hpa") {
		return dto.HPADetail{}, nil
	}
	result, err := kubeResources.GetHPAByName(
		h.Factory.Autoscaling().V2().HorizontalPodAutoscalers().Lister(),
		namespace,
		name,
	)
	if err != nil {
		log.Printf("app: GetHPAByName: %v", err)
		return dto.HPADetail{}, err
	}
	return result, nil
}

func (a *App) ListHPAs() ([]dto.HPA, error) {
	h, namespaces := a.activeFactoryAndNamespaces()
	if !waitForResourceSync(h, "hpa") {
		return []dto.HPA{}, nil
	}
	result, err := kubeResources.ListHPAs(h.Factory.Autoscaling().V2().HorizontalPodAutoscalers().Lister(), namespaces)
	if err != nil {
		log.Printf("app: ListHPAs: %v", err)
		return []dto.HPA{}, nil
	}
	return result, nil
}

func (a *App) emitHPAs() {
	h, namespaces := a.activeFactoryAndNamespaces()
	if !waitForResourceSync(h, "hpa") {
		return
	}
	lister := h.Factory.Autoscaling().V2().HorizontalPodAutoscalers().Lister()
	data, err := kubeResources.ListHPAs(lister, namespaces)
	if err != nil {
		log.Printf("app: emitHPAs: %v", err)
		return
	}
	runtime.EventsEmit(a.ctx, "hpas:update", data)
}

// DeleteHPA deletes an HPA from the specified namespace.
func (a *App) DeleteHPA(namespace, name string) error {
	cs, err := a.activeClientset()
	if err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(context.Background(), apiMutationTimeout)
	defer cancel()
	err = cs.AutoscalingV2().HorizontalPodAutoscalers(namespace).Delete(ctx, name, metav1.DeleteOptions{})
	if err != nil && !errors.IsNotFound(err) {
		return fmt.Errorf("delete HPA: %w", err)
	}

	// Emit update event after successful delete
	a.emitHPAs()

	return nil
}

// DeleteHPAs deletes multiple HPAs, handling best-effort deletion across namespaces.
func (a *App) DeleteHPAs(items []dto.HPARef) error {
	cs, err := a.activeClientset()
	if err != nil {
		return err
	}

	err = deleteRefsBestEffort(items,
		func(r dto.HPARef) string { return r.Namespace },
		func(r dto.HPARef) string { return r.Name },
		"hpas",
		func(ctx context.Context, namespace, name string) error {
			return cs.AutoscalingV2().HorizontalPodAutoscalers(namespace).Delete(ctx, name, metav1.DeleteOptions{})
		},
	)

	a.emitHPAs()

	return err
}

func (a *App) GetHPAYAML(namespace, name string) (string, error) {
	cs, err := a.activeClientset()
	if err != nil {
		return "", err
	}

	ctx, cancel := context.WithTimeout(context.Background(), apiReadTimeout)
	defer cancel()
	hpa, err := cs.AutoscalingV2().HorizontalPodAutoscalers(namespace).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		return "", fmt.Errorf("get HPA: %w", err)
	}

	yamlBytes, err := sigsyaml.Marshal(hpa)
	if err != nil {
		return "", fmt.Errorf("marshal HPA to YAML: %w", err)
	}

	return string(yamlBytes), nil
}

func (a *App) UpdateHPAYAML(namespace, yamlString string) error {
	cs, err := a.activeClientset()
	if err != nil {
		return err
	}

	var hpa autoscalingv2.HorizontalPodAutoscaler
	err = sigsyaml.Unmarshal([]byte(yamlString), &hpa)
	if err != nil {
		return fmt.Errorf("unmarshal YAML to HPA: %w", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), apiMutationTimeout)
	defer cancel()
	_, err = cs.AutoscalingV2().HorizontalPodAutoscalers(namespace).Update(ctx, &hpa, metav1.UpdateOptions{})
	if err != nil {
		return fmt.Errorf("update HPA: %w", err)
	}

	a.emitHPAs()

	return nil
}

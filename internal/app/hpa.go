package app

import (
	"context"
	"fmt"
	"log"
	"strings"

	"github.com/litelensapp/litelens/packages/core/dto"
	"github.com/litelensapp/litelens/internal/kube/resources"
	"github.com/wailsapp/wails/v2/pkg/runtime"
	autoscalingv2 "k8s.io/api/autoscaling/v2"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	sigsyaml "sigs.k8s.io/yaml"
)

func (a *App) GetHPAByName(namespace, name string) (dto.HPADetail, error) {
	a.mu.RLock()
	h := a.factories[a.activeContext]
	a.mu.RUnlock()
	if h == nil {
		return dto.HPADetail{}, nil
	}
	if h.IsForbidden("hpa") {
		return dto.HPADetail{}, nil
	}
	<-h.GetSyncedChan("hpa")
	if h.IsForbidden("hpa") {
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

func (a *App) ListHPAs(namespaces []string) ([]dto.HPA, error) {
	a.mu.RLock()
	h := a.factories[a.activeContext]
	a.mu.RUnlock()
	if h == nil {
		return []dto.HPA{}, nil
	}
	if h.IsForbidden("hpa") {
		return []dto.HPA{}, nil
	}
	<-h.GetSyncedChan("hpa")
	if h.IsForbidden("hpa") {
		return nil, nil
	}
	result, err := kubeResources.ListHPAs(h.Factory.Autoscaling().V2().HorizontalPodAutoscalers().Lister(), namespaces)
	if err != nil {
		log.Printf("app: ListHPAs: %v", err)
		return []dto.HPA{}, nil
	}
	return result, nil
}

func (a *App) emitHPAs(namespaces []string) {
	a.mu.RLock()
	h := a.factories[a.activeContext]
	a.mu.RUnlock()
	if h == nil {
		return
	}
	if h.IsForbidden("hpa") {
		return
	}
	<-h.GetSyncedChan("hpa")
	if h.IsForbidden("hpa") {
		return
	}
	lister := h.Factory.Autoscaling().V2().HorizontalPodAutoscalers().Lister()
	allData, err := kubeResources.ListHPAs(lister, nil)
	if err != nil {
		log.Printf("app: emitHPAs: %v", err)
		return
	}
	runtime.EventsEmit(a.ctx, "hpas:update", allData)
	for _, ns := range namespaces {
		// Filter already-fetched cluster-wide data instead of re-listing
		nsData := make([]dto.HPA, 0)
		for _, item := range allData {
			if item.Namespace == ns {
				nsData = append(nsData, item)
			}
		}
		runtime.EventsEmit(a.ctx, "hpas:"+ns+":update", nsData)
	}
}

// DeleteHPA deletes an HPA from the specified namespace.
func (a *App) DeleteHPA(namespace, name string) error {
	a.mu.RLock()
	cs := a.clients[a.activeContext]
	a.mu.RUnlock()
	if cs == nil {
		return fmt.Errorf("not connected")
	}

	ctx, cancel := context.WithTimeout(context.Background(), apiMutationTimeout)
	defer cancel()
	err := cs.AutoscalingV2().HorizontalPodAutoscalers(namespace).Delete(ctx, name, metav1.DeleteOptions{})
	if err != nil && !errors.IsNotFound(err) {
		return fmt.Errorf("delete HPA: %w", err)
	}

	// Emit update event after successful delete
	a.emitHPAs([]string{namespace})

	return nil
}

// DeleteHPAs deletes multiple HPAs, handling best-effort deletion across namespaces.
func (a *App) DeleteHPAs(items []dto.HPARef) error {
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
		err := cs.AutoscalingV2().HorizontalPodAutoscalers(ref.Namespace).Delete(ctx, ref.Name, metav1.DeleteOptions{})
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
	a.emitHPAs(touchedNamespaces)

	if len(msgs) > 0 {
		return fmt.Errorf("failed to delete %d of %d hpas: %s", len(msgs), len(items), strings.Join(msgs, "; "))
	}
	return nil
}

func (a *App) GetHPAYAML(namespace, name string) (string, error) {
	a.mu.RLock()
	cs := a.clients[a.activeContext]
	a.mu.RUnlock()
	if cs == nil {
		return "", fmt.Errorf("not connected")
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
	a.mu.RLock()
	cs := a.clients[a.activeContext]
	a.mu.RUnlock()
	if cs == nil {
		return fmt.Errorf("not connected")
	}

	var hpa autoscalingv2.HorizontalPodAutoscaler
	err := sigsyaml.Unmarshal([]byte(yamlString), &hpa)
	if err != nil {
		return fmt.Errorf("unmarshal YAML to HPA: %w", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), apiMutationTimeout)
	defer cancel()
	_, err = cs.AutoscalingV2().HorizontalPodAutoscalers(namespace).Update(ctx, &hpa, metav1.UpdateOptions{})
	if err != nil {
		return fmt.Errorf("update HPA: %w", err)
	}

	a.emitHPAs([]string{namespace})

	return nil
}

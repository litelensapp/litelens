package app

import (
	"context"
	"fmt"
	"log"
	"strings"

	"github.com/litelensapp/litelens/internal/dto"
	"github.com/litelensapp/litelens/internal/kube/resources"
	"github.com/wailsapp/wails/v2/pkg/runtime"
	schedulingv1 "k8s.io/api/scheduling/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	sigsyaml "sigs.k8s.io/yaml"
)

func (a *App) GetPriorityClassByName(name string) (dto.PriorityClass, error) {
	a.mu.RLock()
	h := a.factories[a.activeContext]
	a.mu.RUnlock()
	if h == nil {
		return dto.PriorityClass{}, nil
	}
	if h.IsForbidden("priorityclasses") {
		return dto.PriorityClass{}, nil
	}
	<-h.GetSyncedChan("priorityclasses")
	if h.IsForbidden("priorityclasses") {
		return dto.PriorityClass{}, nil
	}
	result, err := kubeResources.GetPriorityClassByName(
		h.Factory.Scheduling().V1().PriorityClasses().Lister(),
		name,
	)
	if err != nil {
		log.Printf("app: GetPriorityClassByName: %v", err)
		return dto.PriorityClass{}, nil
	}
	return result, nil
}

func (a *App) ListPriorityClasses() ([]dto.PriorityClass, error) {
	a.mu.RLock()
	h := a.factories[a.activeContext]
	a.mu.RUnlock()
	if h == nil {
		return []dto.PriorityClass{}, nil
	}
	if h.IsForbidden("priorityclasses") {
		return []dto.PriorityClass{}, nil
	}
	<-h.GetSyncedChan("priorityclasses")
	if h.IsForbidden("priorityclasses") {
		return nil, nil
	}
	result, err := kubeResources.ListPriorityClasses(
		h.Factory.Scheduling().V1().PriorityClasses().Lister(),
	)
	if err != nil {
		log.Printf("app: ListPriorityClasses: %v", err)
		return []dto.PriorityClass{}, nil
	}
	return result, nil
}

func (a *App) DeletePriorityClass(name string) error {
	a.mu.RLock()
	cs := a.clients[a.activeContext]
	a.mu.RUnlock()
	if cs == nil {
		return fmt.Errorf("not connected")
	}

	ctx, cancel := context.WithTimeout(context.Background(), apiMutationTimeout)
	defer cancel()
	err := cs.SchedulingV1().PriorityClasses().Delete(ctx, name, metav1.DeleteOptions{})
	if err != nil && !errors.IsNotFound(err) {
		return fmt.Errorf("delete PriorityClass: %w", err)
	}

	a.emitPriorityClasses()

	return nil
}

func (a *App) DeletePriorityClasses(items []dto.PriorityClassRef) error {
	a.mu.RLock()
	cs := a.clients[a.activeContext]
	a.mu.RUnlock()
	if cs == nil {
		return fmt.Errorf("not connected")
	}

	var msgs []string
	for _, ref := range items {
		ctx, cancel := context.WithTimeout(context.Background(), apiMutationTimeout)
		err := cs.SchedulingV1().PriorityClasses().Delete(ctx, ref.Name, metav1.DeleteOptions{})
		cancel()
		if err != nil && !errors.IsNotFound(err) {
			msgs = append(msgs, fmt.Sprintf("%s: %v", ref.Name, err))
		}
	}

	a.emitPriorityClasses()

	if len(msgs) > 0 {
		return fmt.Errorf("failed to delete %d of %d priorityclasses: %s", len(msgs), len(items), strings.Join(msgs, "; "))
	}
	return nil
}

func (a *App) emitPriorityClasses() {
	a.mu.RLock()
	h := a.factories[a.activeContext]
	a.mu.RUnlock()
	if h == nil {
		return
	}
	if h.IsForbidden("priorityclasses") {
		return
	}
	<-h.GetSyncedChan("priorityclasses")
	if h.IsForbidden("priorityclasses") {
		return
	}
	data, err := kubeResources.ListPriorityClasses(
		h.Factory.Scheduling().V1().PriorityClasses().Lister(),
	)
	if err != nil {
		log.Printf("app: emitPriorityClasses: %v", err)
		return
	}
	runtime.EventsEmit(a.ctx, "priorityclasses:update", data)
}

func (a *App) GetPriorityClassYAML(name string) (string, error) {
	a.mu.RLock()
	cs := a.clients[a.activeContext]
	a.mu.RUnlock()
	if cs == nil {
		return "", fmt.Errorf("not connected")
	}

	ctx, cancel := context.WithTimeout(context.Background(), apiReadTimeout)
	defer cancel()
	pc, err := cs.SchedulingV1().PriorityClasses().Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		return "", fmt.Errorf("get PriorityClass: %w", err)
	}

	yamlBytes, err := sigsyaml.Marshal(pc)
	if err != nil {
		return "", fmt.Errorf("marshal PriorityClass to YAML: %w", err)
	}

	return string(yamlBytes), nil
}

func (a *App) UpdatePriorityClassYAML(yamlString string) error {
	a.mu.RLock()
	cs := a.clients[a.activeContext]
	a.mu.RUnlock()
	if cs == nil {
		return fmt.Errorf("not connected")
	}

	var pc schedulingv1.PriorityClass
	err := sigsyaml.Unmarshal([]byte(yamlString), &pc)
	if err != nil {
		return fmt.Errorf("unmarshal YAML to PriorityClass: %w", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), apiMutationTimeout)
	defer cancel()
	_, err = cs.SchedulingV1().PriorityClasses().Update(ctx, &pc, metav1.UpdateOptions{})
	if err != nil {
		return fmt.Errorf("update PriorityClass: %w", err)
	}

	a.emitPriorityClasses()

	return nil
}

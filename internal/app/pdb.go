package app

import (
	"context"
	"fmt"
	"log"
	"strings"

	"github.com/litelensapp/litelens/internal/kube/resources"
	"github.com/litelensapp/litelens/packages/core/dto"
	"github.com/wailsapp/wails/v2/pkg/runtime"
	policyv1 "k8s.io/api/policy/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	sigsyaml "sigs.k8s.io/yaml"
)

func (a *App) ListPodDisruptionBudgets() ([]dto.PodDisruptionBudget, error) {
	a.mu.RLock()
	h := a.factories[a.activeContext]
	namespaces := a.activeNamespaces
	a.mu.RUnlock()
	if h == nil {
		return []dto.PodDisruptionBudget{}, nil
	}
	if h.IsForbidden("pdbs") {
		return []dto.PodDisruptionBudget{}, nil
	}
	<-h.GetSyncedChan("pdbs")
	if h.IsForbidden("pdbs") {
		return nil, nil
	}
	result, err := kubeResources.ListPodDisruptionBudgets(h.Factory.Policy().V1().PodDisruptionBudgets().Lister(), namespaces)
	if err != nil {
		log.Printf("app: ListPodDisruptionBudgets: %v", err)
		return []dto.PodDisruptionBudget{}, nil
	}
	return result, nil
}

func (a *App) GetPodDisruptionBudgetByName(namespace, name string) (*dto.PodDisruptionBudgetDetail, error) {
	a.mu.RLock()
	h := a.factories[a.activeContext]
	a.mu.RUnlock()
	if h == nil {
		return &dto.PodDisruptionBudgetDetail{}, nil
	}
	if h.IsForbidden("pdbs") {
		return &dto.PodDisruptionBudgetDetail{}, nil
	}
	<-h.GetSyncedChan("pdbs")
	if h.IsForbidden("pdbs") {
		return nil, nil
	}
	result, err := kubeResources.GetPodDisruptionBudgetByName(
		h.Factory.Policy().V1().PodDisruptionBudgets().Lister(),
		namespace,
		name,
	)
	if err != nil {
		log.Printf("app: GetPodDisruptionBudgetByName: %v", err)
		return &dto.PodDisruptionBudgetDetail{}, nil
	}
	return result, nil
}

func (a *App) emitPodDisruptionBudgets() {
	a.mu.RLock()
	h := a.factories[a.activeContext]
	namespaces := a.activeNamespaces
	a.mu.RUnlock()
	if h == nil {
		return
	}
	if h.IsForbidden("pdbs") {
		return
	}
	<-h.GetSyncedChan("pdbs")
	if h.IsForbidden("pdbs") {
		return
	}
	lister := h.Factory.Policy().V1().PodDisruptionBudgets().Lister()
	data, err := kubeResources.ListPodDisruptionBudgets(lister, namespaces)
	if err != nil {
		log.Printf("app: emitPodDisruptionBudgets: %v", err)
		return
	}
	runtime.EventsEmit(a.ctx, "pdbs:update", data)
}

// DeletePodDisruptionBudget deletes a PodDisruptionBudget from the specified namespace.
func (a *App) DeletePodDisruptionBudget(namespace, name string) error {
	a.mu.RLock()
	cs := a.clients[a.activeContext]
	a.mu.RUnlock()
	if cs == nil {
		return fmt.Errorf("not connected")
	}

	ctx, cancel := context.WithTimeout(context.Background(), apiMutationTimeout)
	defer cancel()
	err := cs.PolicyV1().PodDisruptionBudgets(namespace).Delete(ctx, name, metav1.DeleteOptions{})
	if err != nil && !errors.IsNotFound(err) {
		return fmt.Errorf("delete PodDisruptionBudget: %w", err)
	}

	// Emit update event after successful delete
	a.emitPodDisruptionBudgets()

	return nil
}

// DeletePodDisruptionBudgets deletes multiple PodDisruptionBudgets, handling best-effort deletion across namespaces.
func (a *App) DeletePodDisruptionBudgets(items []dto.PodDisruptionBudgetRef) error {
	a.mu.RLock()
	cs := a.clients[a.activeContext]
	a.mu.RUnlock()
	if cs == nil {
		return fmt.Errorf("not connected")
	}

	var msgs []string

	for _, ref := range items {
		ctx, cancel := context.WithTimeout(context.Background(), apiMutationTimeout)
		err := cs.PolicyV1().PodDisruptionBudgets(ref.Namespace).Delete(ctx, ref.Name, metav1.DeleteOptions{})
		cancel()
		if err != nil && !errors.IsNotFound(err) {
			msgs = append(msgs, fmt.Sprintf("%s/%s: %v", ref.Namespace, ref.Name, err))
		}
	}

	a.emitPodDisruptionBudgets()

	if len(msgs) > 0 {
		return fmt.Errorf("failed to delete %d of %d poddisruptionbudgets: %s", len(msgs), len(items), strings.Join(msgs, "; "))
	}
	return nil
}

func (a *App) GetPDBYAML(namespace, name string) (string, error) {
	a.mu.RLock()
	cs := a.clients[a.activeContext]
	a.mu.RUnlock()
	if cs == nil {
		return "", fmt.Errorf("not connected")
	}

	ctx, cancel := context.WithTimeout(context.Background(), apiReadTimeout)
	defer cancel()
	pdb, err := cs.PolicyV1().PodDisruptionBudgets(namespace).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		return "", fmt.Errorf("get PodDisruptionBudget: %w", err)
	}

	yamlBytes, err := sigsyaml.Marshal(pdb)
	if err != nil {
		return "", fmt.Errorf("marshal PodDisruptionBudget to YAML: %w", err)
	}

	return string(yamlBytes), nil
}

func (a *App) UpdatePDBYAML(namespace, yamlString string) error {
	a.mu.RLock()
	cs := a.clients[a.activeContext]
	a.mu.RUnlock()
	if cs == nil {
		return fmt.Errorf("not connected")
	}

	var pdb policyv1.PodDisruptionBudget
	err := sigsyaml.Unmarshal([]byte(yamlString), &pdb)
	if err != nil {
		return fmt.Errorf("unmarshal YAML to PodDisruptionBudget: %w", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), apiMutationTimeout)
	defer cancel()
	_, err = cs.PolicyV1().PodDisruptionBudgets(namespace).Update(ctx, &pdb, metav1.UpdateOptions{})
	if err != nil {
		return fmt.Errorf("update PodDisruptionBudget: %w", err)
	}

	a.emitPodDisruptionBudgets()

	return nil
}

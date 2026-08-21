package app

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/litelensapp/litelens/packages/core/dto"
	"github.com/litelensapp/litelens/internal/kube/resources"
	"github.com/wailsapp/wails/v2/pkg/runtime"
	appsv1 "k8s.io/api/apps/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/types"
	sigsyaml "sigs.k8s.io/yaml"
)

func (a *App) ListDeployments(namespaces []string) ([]dto.Deployment, error) {
	a.mu.RLock()
	h := a.factories[a.activeContext]
	a.mu.RUnlock()
	if h == nil {
		return []dto.Deployment{}, nil
	}
	if h.IsForbidden("deployments") {
		return []dto.Deployment{}, nil
	}
	<-h.GetSyncedChan("deployments")
	if h.IsForbidden("deployments") {
		return nil, nil
	}
	result, err := kubeResources.ListDeployments(h.Factory.Apps().V1().Deployments().Lister(), namespaces)
	if err != nil {
		log.Printf("app: ListDeployments: %v", err)
		return []dto.Deployment{}, nil
	}
	return result, nil
}

func (a *App) GetDeploymentByName(namespace, name string) (dto.Deployment, error) {
	a.mu.RLock()
	h := a.factories[a.activeContext]
	a.mu.RUnlock()
	if h == nil {
		return dto.Deployment{}, nil
	}
	if h.IsForbidden("deployments") {
		return dto.Deployment{}, nil
	}
	<-h.GetSyncedChan("deployments")
	if h.IsForbidden("deployments") {
		return dto.Deployment{}, nil
	}
	result, err := kubeResources.GetDeploymentByName(h.Factory.Apps().V1().Deployments().Lister(), namespace, name)
	if err != nil {
		log.Printf("app: GetDeploymentByName: %v", err)
		return dto.Deployment{}, nil
	}
	return result, nil
}

func (a *App) GetDeploymentsSummary(namespace string) (dto.DeploymentSummary, error) {
	a.mu.RLock()
	h := a.factories[a.activeContext]
	a.mu.RUnlock()
	if h == nil {
		return dto.DeploymentSummary{}, nil
	}
	if h.IsForbidden("deployments") {
		return dto.DeploymentSummary{}, nil
	}
	<-h.GetSyncedChan("deployments")
	if h.IsForbidden("deployments") {
		return dto.DeploymentSummary{}, nil
	}
	var deps []*appsv1.Deployment
	var err error
	lister := h.Factory.Apps().V1().Deployments().Lister()
	if namespace == "" {
		deps, err = lister.List(labels.Everything())
	} else {
		deps, err = lister.Deployments(namespace).List(labels.Everything())
	}
	if err != nil {
		log.Printf("app: GetDeploymentsSummary: %v", err)
		return dto.DeploymentSummary{}, nil
	}
	return kubeResources.SummarizeDeployments(deps), nil
}

func (a *App) RestartDeployment(namespace, name string) error {
	a.mu.RLock()
	cs := a.clients[a.activeContext]
	a.mu.RUnlock()
	if cs == nil {
		return fmt.Errorf("not connected")
	}
	patchBody := map[string]any{
		"spec": map[string]any{
			"template": map[string]any{
				"metadata": map[string]any{
					"annotations": map[string]any{
						"kubectl.kubernetes.io/restartedAt": time.Now().UTC().Format(time.RFC3339),
					},
				},
			},
		},
	}
	patchBytes, err := json.Marshal(patchBody)
	if err != nil {
		return err
	}
	ctx, cancel := context.WithTimeout(context.Background(), apiMutationTimeout)
	defer cancel()
	_, err = cs.AppsV1().Deployments(namespace).Patch(
		ctx, name, types.MergePatchType, patchBytes, metav1.PatchOptions{},
	)
	return err
}

func (a *App) ScaleDeployment(namespace, name string, replicas int32) error {
	a.mu.RLock()
	cs := a.clients[a.activeContext]
	a.mu.RUnlock()
	if cs == nil {
		return fmt.Errorf("not connected")
	}
	patchBody := map[string]any{
		"spec": map[string]any{"replicas": replicas},
	}
	patchBytes, err := json.Marshal(patchBody)
	if err != nil {
		return err
	}
	ctx, cancel := context.WithTimeout(context.Background(), apiMutationTimeout)
	defer cancel()
	_, err = cs.AppsV1().Deployments(namespace).Patch(
		ctx, name, types.MergePatchType, patchBytes, metav1.PatchOptions{},
	)
	return err
}

func (a *App) DeleteDeployment(namespace, name string) error {
	a.mu.RLock()
	cs := a.clients[a.activeContext]
	a.mu.RUnlock()
	if cs == nil {
		return fmt.Errorf("not connected")
	}

	ctx, cancel := context.WithTimeout(context.Background(), apiMutationTimeout)
	defer cancel()
	err := cs.AppsV1().Deployments(namespace).Delete(ctx, name, metav1.DeleteOptions{})
	if err != nil && !errors.IsNotFound(err) {
		return fmt.Errorf("delete Deployment: %w", err)
	}

	a.emitDeployments([]string{namespace})

	return nil
}

// DeleteDeployments deletes multiple Deployments, handling best-effort deletion across namespaces.
func (a *App) DeleteDeployments(items []dto.DeploymentRef) error {
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
		err := cs.AppsV1().Deployments(ref.Namespace).Delete(ctx, ref.Name, metav1.DeleteOptions{})
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
	a.emitDeployments(touchedNamespaces)

	if len(msgs) > 0 {
		return fmt.Errorf("failed to delete %d of %d deployments: %s", len(msgs), len(items), strings.Join(msgs, "; "))
	}
	return nil
}

func (a *App) emitDeployments(namespaces []string) {
	a.mu.RLock()
	h := a.factories[a.activeContext]
	a.mu.RUnlock()
	if h == nil {
		return
	}
	if h.IsForbidden("deployments") {
		return
	}
	<-h.GetSyncedChan("deployments")
	if h.IsForbidden("deployments") {
		return
	}
	lister := h.Factory.Apps().V1().Deployments().Lister()
	allData, err := kubeResources.ListDeployments(lister, nil)
	if err != nil {
		log.Printf("app: emitDeployments: %v", err)
		return
	}
	runtime.EventsEmit(a.ctx, "deployments:update", allData)
	for _, ns := range namespaces {
		// Filter already-fetched cluster-wide data instead of re-listing
		nsData := make([]dto.Deployment, 0)
		for _, d := range allData {
			if d.Namespace == ns {
				nsData = append(nsData, d)
			}
		}
		runtime.EventsEmit(a.ctx, "deployments:"+ns+":update", nsData)
	}
}

func (a *App) GetDeploymentYAML(namespace, name string) (string, error) {
	a.mu.RLock()
	cs := a.clients[a.activeContext]
	a.mu.RUnlock()
	if cs == nil {
		return "", fmt.Errorf("not connected")
	}

	ctx, cancel := context.WithTimeout(context.Background(), apiReadTimeout)
	defer cancel()
	dep, err := cs.AppsV1().Deployments(namespace).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		return "", fmt.Errorf("get Deployment: %w", err)
	}

	yamlBytes, err := sigsyaml.Marshal(dep)
	if err != nil {
		return "", fmt.Errorf("marshal Deployment to YAML: %w", err)
	}

	return string(yamlBytes), nil
}

func (a *App) UpdateDeploymentYAML(namespace, yamlString string) error {
	a.mu.RLock()
	cs := a.clients[a.activeContext]
	a.mu.RUnlock()
	if cs == nil {
		return fmt.Errorf("not connected")
	}

	var dep appsv1.Deployment
	err := sigsyaml.Unmarshal([]byte(yamlString), &dep)
	if err != nil {
		return fmt.Errorf("unmarshal YAML to Deployment: %w", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), apiMutationTimeout)
	defer cancel()
	_, err = cs.AppsV1().Deployments(namespace).Update(ctx, &dep, metav1.UpdateOptions{})
	if err != nil {
		return fmt.Errorf("update Deployment: %w", err)
	}

	a.emitDeployments([]string{namespace})

	return nil
}

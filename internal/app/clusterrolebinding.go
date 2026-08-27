package app

import (
	"context"
	"fmt"
	"log"
	"strings"

	"github.com/litelensapp/litelens/packages/core/kube/dto"
	"github.com/litelensapp/litelens/internal/kube/resources"
	"github.com/wailsapp/wails/v2/pkg/runtime"
	rbacv1 "k8s.io/api/rbac/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	sigsyaml "sigs.k8s.io/yaml"
)

func (a *App) GetClusterRoleBindingByName(name string) (dto.ClusterRoleBinding, error) {
	a.mu.RLock()
	h := a.factories[a.activeContext]
	a.mu.RUnlock()
	if h == nil {
		return dto.ClusterRoleBinding{}, nil
	}
	if h.IsForbidden("clusterrolebindings") {
		return dto.ClusterRoleBinding{}, nil
	}
	<-h.GetSyncedChan("clusterrolebindings")
	if h.IsForbidden("clusterrolebindings") {
		return dto.ClusterRoleBinding{}, nil
	}
	result, err := kubeResources.GetClusterRoleBindingByName(
		h.Factory.Rbac().V1().ClusterRoleBindings().Lister(),
		name,
	)
	if err != nil {
		log.Printf("app: GetClusterRoleBindingByName: %v", err)
		return dto.ClusterRoleBinding{}, nil
	}
	return result, nil
}

func (a *App) ListClusterRoleBindings() ([]dto.ClusterRoleBinding, error) {
	a.mu.RLock()
	h := a.factories[a.activeContext]
	a.mu.RUnlock()
	if h == nil {
		return []dto.ClusterRoleBinding{}, nil
	}
	if h.IsForbidden("clusterrolebindings") {
		return []dto.ClusterRoleBinding{}, nil
	}
	<-h.GetSyncedChan("clusterrolebindings")
	if h.IsForbidden("clusterrolebindings") {
		return []dto.ClusterRoleBinding{}, nil
	}
	result, err := kubeResources.ListClusterRoleBindings(
		h.Factory.Rbac().V1().ClusterRoleBindings().Lister(),
	)
	if err != nil {
		log.Printf("app: ListClusterRoleBindings: %v", err)
		return []dto.ClusterRoleBinding{}, nil
	}
	return result, nil
}

func (a *App) emitClusterRoleBindings() {
	a.mu.RLock()
	h := a.factories[a.activeContext]
	a.mu.RUnlock()
	if h == nil {
		return
	}
	if h.IsForbidden("clusterrolebindings") {
		return
	}
	data, err := kubeResources.ListClusterRoleBindings(
		h.Factory.Rbac().V1().ClusterRoleBindings().Lister(),
	)
	if err != nil {
		log.Printf("app: emitClusterRoleBindings: %v", err)
		return
	}
	runtime.EventsEmit(a.ctx, "clusterrolebindings:update", data)
}

func (a *App) DeleteClusterRoleBinding(name string) error {
	a.mu.RLock()
	cs := a.clients[a.activeContext]
	a.mu.RUnlock()
	if cs == nil {
		return fmt.Errorf("not connected")
	}

	ctx, cancel := context.WithTimeout(context.Background(), apiMutationTimeout)
	defer cancel()
	err := cs.RbacV1().ClusterRoleBindings().Delete(ctx, name, metav1.DeleteOptions{})
	if err != nil && !errors.IsNotFound(err) {
		return fmt.Errorf("delete ClusterRoleBinding: %w", err)
	}

	a.emitClusterRoleBindings()

	return nil
}

func (a *App) DeleteClusterRoleBindings(items []dto.ClusterRoleBindingRef) error {
	a.mu.RLock()
	cs := a.clients[a.activeContext]
	a.mu.RUnlock()
	if cs == nil {
		return fmt.Errorf("not connected")
	}

	var msgs []string

	for _, ref := range items {
		ctx, cancel := context.WithTimeout(context.Background(), apiMutationTimeout)
		err := cs.RbacV1().ClusterRoleBindings().Delete(ctx, ref.Name, metav1.DeleteOptions{})
		cancel()
		if err != nil && !errors.IsNotFound(err) {
			msgs = append(msgs, fmt.Sprintf("%s: %v", ref.Name, err))
		}
	}

	a.emitClusterRoleBindings()

	if len(msgs) > 0 {
		return fmt.Errorf("failed to delete %d of %d clusterrolebindings: %s", len(msgs), len(items), strings.Join(msgs, "; "))
	}
	return nil
}

func (a *App) GetClusterRoleBindingYAML(name string) (string, error) {
	a.mu.RLock()
	cs := a.clients[a.activeContext]
	a.mu.RUnlock()
	if cs == nil {
		return "", fmt.Errorf("not connected")
	}

	ctx, cancel := context.WithTimeout(context.Background(), apiReadTimeout)
	defer cancel()
	crb, err := cs.RbacV1().ClusterRoleBindings().Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		return "", fmt.Errorf("get ClusterRoleBinding: %w", err)
	}

	b, err := sigsyaml.Marshal(crb)
	if err != nil {
		return "", fmt.Errorf("marshal ClusterRoleBinding: %w", err)
	}

	return string(b), nil
}

func (a *App) UpdateClusterRoleBindingYAML(yamlString string) error {
	a.mu.RLock()
	cs := a.clients[a.activeContext]
	a.mu.RUnlock()
	if cs == nil {
		return fmt.Errorf("not connected")
	}

	var crb rbacv1.ClusterRoleBinding
	err := sigsyaml.Unmarshal([]byte(yamlString), &crb)
	if err != nil {
		return fmt.Errorf("unmarshal YAML to ClusterRoleBinding: %w", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), apiMutationTimeout)
	defer cancel()
	_, err = cs.RbacV1().ClusterRoleBindings().Update(ctx, &crb, metav1.UpdateOptions{})
	if err != nil {
		return fmt.Errorf("update ClusterRoleBinding: %w", err)
	}

	a.emitClusterRoleBindings()

	return nil
}

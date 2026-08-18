package app

import (
	"context"
	"fmt"
	"log"
	"strings"

	"github.com/litelensapp/litelens/packages/core/dto"
	kubeResources "github.com/litelensapp/litelens/internal/kube/resources"
	"github.com/wailsapp/wails/v2/pkg/runtime"
	rbacv1 "k8s.io/api/rbac/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	sigsyaml "sigs.k8s.io/yaml"
)

func (a *App) GetClusterRoleByName(name string) (dto.ClusterRole, error) {
	a.mu.RLock()
	h := a.factories[a.activeContext]
	a.mu.RUnlock()
	if h == nil {
		return dto.ClusterRole{}, nil
	}
	if h.IsForbidden("clusterroles") {
		return dto.ClusterRole{}, nil
	}
	<-h.GetSyncedChan("clusterroles")
	if h.IsForbidden("clusterroles") {
		return dto.ClusterRole{}, nil
	}
	result, err := kubeResources.GetClusterRoleByName(
		h.Factory.Rbac().V1().ClusterRoles().Lister(),
		name,
	)
	if err != nil {
		log.Printf("app: GetClusterRoleByName: %v", err)
		return dto.ClusterRole{}, nil
	}
	return result, nil
}

func (a *App) ListClusterRoles() ([]dto.ClusterRole, error) {
	a.mu.RLock()
	h := a.factories[a.activeContext]
	a.mu.RUnlock()
	if h == nil {
		return []dto.ClusterRole{}, nil
	}
	if h.IsForbidden("clusterroles") {
		return []dto.ClusterRole{}, nil
	}
	<-h.GetSyncedChan("clusterroles")
	if h.IsForbidden("clusterroles") {
		return []dto.ClusterRole{}, nil
	}
	result, err := kubeResources.ListClusterRoles(
		h.Factory.Rbac().V1().ClusterRoles().Lister(),
	)
	if err != nil {
		log.Printf("app: ListClusterRoles: %v", err)
		return []dto.ClusterRole{}, nil
	}
	return result, nil
}

func (a *App) emitClusterRoles() {
	a.mu.RLock()
	h := a.factories[a.activeContext]
	a.mu.RUnlock()
	if h == nil {
		return
	}
	if h.IsForbidden("clusterroles") {
		return
	}
	data, err := kubeResources.ListClusterRoles(
		h.Factory.Rbac().V1().ClusterRoles().Lister(),
	)
	if err != nil {
		log.Printf("app: emitClusterRoles: %v", err)
		return
	}
	runtime.EventsEmit(a.ctx, "clusterroles:update", data)
}

func (a *App) DeleteClusterRole(name string) error {
	a.mu.RLock()
	cs := a.clients[a.activeContext]
	a.mu.RUnlock()
	if cs == nil {
		return fmt.Errorf("not connected")
	}

	ctx, cancel := context.WithTimeout(context.Background(), apiMutationTimeout)
	defer cancel()
	err := cs.RbacV1().ClusterRoles().Delete(ctx, name, metav1.DeleteOptions{})
	if err != nil && !errors.IsNotFound(err) {
		return fmt.Errorf("delete ClusterRole: %w", err)
	}

	a.emitClusterRoles()

	return nil
}

func (a *App) DeleteClusterRoles(items []dto.ClusterRoleRef) error {
	a.mu.RLock()
	cs := a.clients[a.activeContext]
	a.mu.RUnlock()
	if cs == nil {
		return fmt.Errorf("not connected")
	}

	var msgs []string

	for _, ref := range items {
		ctx, cancel := context.WithTimeout(context.Background(), apiMutationTimeout)
		err := cs.RbacV1().ClusterRoles().Delete(ctx, ref.Name, metav1.DeleteOptions{})
		cancel()
		if err != nil && !errors.IsNotFound(err) {
			msgs = append(msgs, fmt.Sprintf("%s: %v", ref.Name, err))
		}
	}

	a.emitClusterRoles()

	if len(msgs) > 0 {
		return fmt.Errorf("failed to delete %d of %d clusterroles: %s", len(msgs), len(items), strings.Join(msgs, "; "))
	}
	return nil
}

func (a *App) GetClusterRoleYAML(name string) (string, error) {
	a.mu.RLock()
	cs := a.clients[a.activeContext]
	a.mu.RUnlock()
	if cs == nil {
		return "", fmt.Errorf("not connected")
	}

	ctx, cancel := context.WithTimeout(context.Background(), apiReadTimeout)
	defer cancel()
	cr, err := cs.RbacV1().ClusterRoles().Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		return "", fmt.Errorf("get ClusterRole: %w", err)
	}

	b, err := sigsyaml.Marshal(cr)
	if err != nil {
		return "", fmt.Errorf("marshal ClusterRole: %w", err)
	}

	return string(b), nil
}

func (a *App) UpdateClusterRoleYAML(yamlString string) error {
	a.mu.RLock()
	cs := a.clients[a.activeContext]
	a.mu.RUnlock()
	if cs == nil {
		return fmt.Errorf("not connected")
	}

	var cr rbacv1.ClusterRole
	err := sigsyaml.Unmarshal([]byte(yamlString), &cr)
	if err != nil {
		return fmt.Errorf("unmarshal YAML to ClusterRole: %w", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), apiMutationTimeout)
	defer cancel()
	_, err = cs.RbacV1().ClusterRoles().Update(ctx, &cr, metav1.UpdateOptions{})
	if err != nil {
		return fmt.Errorf("update ClusterRole: %w", err)
	}

	a.emitClusterRoles()

	return nil
}

package app

import (
	"context"
	"fmt"
	"log"
	"strings"

	"github.com/litelensapp/litelens/internal/kube/resources"
	"github.com/litelensapp/litelens/packages/core/kube/dto"
	"github.com/wailsapp/wails/v2/pkg/runtime"
	rbacv1 "k8s.io/api/rbac/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	sigsyaml "sigs.k8s.io/yaml"
)

func (a *App) GetRoleByName(namespace, name string) (dto.Role, error) {
	a.mu.RLock()
	h := a.factories[a.activeContext]
	a.mu.RUnlock()
	if h == nil {
		return dto.Role{}, nil
	}
	if h.IsForbidden("roles") {
		return dto.Role{}, nil
	}
	<-h.GetSyncedChan("roles")
	if h.IsForbidden("roles") {
		return dto.Role{}, nil
	}
	result, err := kubeResources.GetRoleByName(
		h.Factory.Rbac().V1().Roles().Lister(),
		namespace,
		name,
	)
	if err != nil {
		log.Printf("app: GetRoleByName: %v", err)
		return dto.Role{}, nil
	}
	return result, nil
}

func (a *App) ListRoles() ([]dto.Role, error) {
	a.mu.RLock()
	h := a.factories[a.activeContext]
	namespaces := a.activeNamespaces
	a.mu.RUnlock()
	if h == nil {
		return []dto.Role{}, nil
	}
	if h.IsForbidden("roles") {
		return []dto.Role{}, nil
	}
	<-h.GetSyncedChan("roles")
	if h.IsForbidden("roles") {
		return nil, nil
	}
	result, err := kubeResources.ListRoles(
		h.Factory.Rbac().V1().Roles().Lister(),
		namespaces,
	)
	if err != nil {
		log.Printf("app: ListRoles: %v", err)
		return []dto.Role{}, nil
	}
	return result, nil
}

func (a *App) emitRoles() {
	a.mu.RLock()
	h := a.factories[a.activeContext]
	namespaces := a.activeNamespaces
	a.mu.RUnlock()
	if h == nil {
		return
	}
	if h.IsForbidden("roles") {
		return
	}
	<-h.GetSyncedChan("roles")
	if h.IsForbidden("roles") {
		return
	}
	lister := h.Factory.Rbac().V1().Roles().Lister()
	data, err := kubeResources.ListRoles(lister, namespaces)
	if err != nil {
		log.Printf("app: emitRoles: %v", err)
		return
	}
	runtime.EventsEmit(a.ctx, "roles:update", data)
}

func (a *App) DeleteRole(namespace, name string) error {
	a.mu.RLock()
	cs := a.clients[a.activeContext]
	a.mu.RUnlock()
	if cs == nil {
		return fmt.Errorf("not connected")
	}

	ctx, cancel := context.WithTimeout(context.Background(), apiMutationTimeout)
	defer cancel()
	err := cs.RbacV1().Roles(namespace).Delete(ctx, name, metav1.DeleteOptions{})
	if err != nil && !errors.IsNotFound(err) {
		return fmt.Errorf("delete Role: %w", err)
	}

	a.emitRoles()

	return nil
}

func (a *App) DeleteRoles(items []dto.RoleRef) error {
	a.mu.RLock()
	cs := a.clients[a.activeContext]
	a.mu.RUnlock()
	if cs == nil {
		return fmt.Errorf("not connected")
	}

	var msgs []string

	for _, ref := range items {
		ctx, cancel := context.WithTimeout(context.Background(), apiMutationTimeout)
		err := cs.RbacV1().Roles(ref.Namespace).Delete(ctx, ref.Name, metav1.DeleteOptions{})
		cancel()
		if err != nil && !errors.IsNotFound(err) {
			msgs = append(msgs, fmt.Sprintf("%s/%s: %v", ref.Namespace, ref.Name, err))
		}
	}

	a.emitRoles()

	if len(msgs) > 0 {
		return fmt.Errorf("failed to delete %d of %d roles: %s", len(msgs), len(items), strings.Join(msgs, "; "))
	}
	return nil
}

func (a *App) GetRoleYAML(namespace, name string) (string, error) {
	a.mu.RLock()
	cs := a.clients[a.activeContext]
	a.mu.RUnlock()
	if cs == nil {
		return "", fmt.Errorf("not connected")
	}

	ctx, cancel := context.WithTimeout(context.Background(), apiReadTimeout)
	defer cancel()
	role, err := cs.RbacV1().Roles(namespace).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		return "", fmt.Errorf("get Role: %w", err)
	}

	b, err := sigsyaml.Marshal(role)
	if err != nil {
		return "", fmt.Errorf("marshal Role: %w", err)
	}

	return string(b), nil
}

func (a *App) UpdateRoleYAML(namespace, yamlString string) error {
	a.mu.RLock()
	cs := a.clients[a.activeContext]
	a.mu.RUnlock()
	if cs == nil {
		return fmt.Errorf("not connected")
	}

	var role rbacv1.Role
	err := sigsyaml.Unmarshal([]byte(yamlString), &role)
	if err != nil {
		return fmt.Errorf("unmarshal YAML to Role: %w", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), apiMutationTimeout)
	defer cancel()
	_, err = cs.RbacV1().Roles(namespace).Update(ctx, &role, metav1.UpdateOptions{})
	if err != nil {
		return fmt.Errorf("update Role: %w", err)
	}

	a.emitRoles()

	return nil
}

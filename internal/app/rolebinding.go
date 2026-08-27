package app

import (
	"context"
	"fmt"
	"log"
	"strings"

	kubeResources "github.com/litelensapp/litelens/internal/kube/resources"
	"github.com/litelensapp/litelens/packages/core/kube/dto"
	"github.com/wailsapp/wails/v2/pkg/runtime"
	rbacv1 "k8s.io/api/rbac/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	sigsyaml "sigs.k8s.io/yaml"
)

func (a *App) GetRoleBindingByName(namespace, name string) (dto.RoleBinding, error) {
	a.mu.RLock()
	h := a.factories[a.activeContext]
	a.mu.RUnlock()
	if h == nil {
		return dto.RoleBinding{}, nil
	}
	if h.IsForbidden("rolebindings") {
		return dto.RoleBinding{}, nil
	}
	<-h.GetSyncedChan("rolebindings")
	if h.IsForbidden("rolebindings") {
		return dto.RoleBinding{}, nil
	}
	result, err := kubeResources.GetRoleBindingByName(
		h.Factory.Rbac().V1().RoleBindings().Lister(),
		namespace,
		name,
	)
	if err != nil {
		log.Printf("app: GetRoleBindingByName: %v", err)
		return dto.RoleBinding{}, nil
	}
	return result, nil
}

func (a *App) ListRoleBindings() ([]dto.RoleBinding, error) {
	a.mu.RLock()
	h := a.factories[a.activeContext]
	namespaces := a.activeNamespaces
	a.mu.RUnlock()
	if h == nil {
		return []dto.RoleBinding{}, nil
	}
	if h.IsForbidden("rolebindings") {
		return []dto.RoleBinding{}, nil
	}
	<-h.GetSyncedChan("rolebindings")
	if h.IsForbidden("rolebindings") {
		return nil, nil
	}
	result, err := kubeResources.ListRoleBindings(
		h.Factory.Rbac().V1().RoleBindings().Lister(),
		namespaces,
	)
	if err != nil {
		log.Printf("app: ListRoleBindings: %v", err)
		return []dto.RoleBinding{}, nil
	}
	return result, nil
}

func (a *App) emitRoleBindings() {
	a.mu.RLock()
	h := a.factories[a.activeContext]
	namespaces := a.activeNamespaces
	a.mu.RUnlock()
	if h == nil {
		return
	}
	if h.IsForbidden("rolebindings") {
		return
	}
	<-h.GetSyncedChan("rolebindings")
	if h.IsForbidden("rolebindings") {
		return
	}
	lister := h.Factory.Rbac().V1().RoleBindings().Lister()
	data, err := kubeResources.ListRoleBindings(lister, namespaces)
	if err != nil {
		log.Printf("app: emitRoleBindings: %v", err)
		return
	}
	runtime.EventsEmit(a.ctx, "rolebindings:update", data)
}

func (a *App) DeleteRoleBinding(namespace, name string) error {
	a.mu.RLock()
	cs := a.clients[a.activeContext]
	a.mu.RUnlock()
	if cs == nil {
		return fmt.Errorf("not connected")
	}

	ctx, cancel := context.WithTimeout(context.Background(), apiMutationTimeout)
	defer cancel()
	err := cs.RbacV1().RoleBindings(namespace).Delete(ctx, name, metav1.DeleteOptions{})
	if err != nil && !errors.IsNotFound(err) {
		return fmt.Errorf("delete RoleBinding: %w", err)
	}

	a.emitRoleBindings()

	return nil
}

func (a *App) DeleteRoleBindings(items []dto.RoleBindingRef) error {
	a.mu.RLock()
	cs := a.clients[a.activeContext]
	a.mu.RUnlock()
	if cs == nil {
		return fmt.Errorf("not connected")
	}

	var msgs []string

	for _, ref := range items {
		ctx, cancel := context.WithTimeout(context.Background(), apiMutationTimeout)
		err := cs.RbacV1().RoleBindings(ref.Namespace).Delete(ctx, ref.Name, metav1.DeleteOptions{})
		cancel()
		if err != nil && !errors.IsNotFound(err) {
			msgs = append(msgs, fmt.Sprintf("%s/%s: %v", ref.Namespace, ref.Name, err))
		}
	}

	a.emitRoleBindings()

	if len(msgs) > 0 {
		return fmt.Errorf("failed to delete %d of %d rolebindings: %s", len(msgs), len(items), strings.Join(msgs, "; "))
	}
	return nil
}

func (a *App) GetRoleBindingYAML(namespace, name string) (string, error) {
	a.mu.RLock()
	cs := a.clients[a.activeContext]
	a.mu.RUnlock()
	if cs == nil {
		return "", fmt.Errorf("not connected")
	}

	ctx, cancel := context.WithTimeout(context.Background(), apiReadTimeout)
	defer cancel()
	rb, err := cs.RbacV1().RoleBindings(namespace).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		return "", fmt.Errorf("get RoleBinding: %w", err)
	}

	b, err := sigsyaml.Marshal(rb)
	if err != nil {
		return "", fmt.Errorf("marshal RoleBinding: %w", err)
	}

	return string(b), nil
}

func (a *App) UpdateRoleBindingYAML(namespace, yamlString string) error {
	a.mu.RLock()
	cs := a.clients[a.activeContext]
	a.mu.RUnlock()
	if cs == nil {
		return fmt.Errorf("not connected")
	}

	var rb rbacv1.RoleBinding
	err := sigsyaml.Unmarshal([]byte(yamlString), &rb)
	if err != nil {
		return fmt.Errorf("unmarshal YAML to RoleBinding: %w", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), apiMutationTimeout)
	defer cancel()
	_, err = cs.RbacV1().RoleBindings(namespace).Update(ctx, &rb, metav1.UpdateOptions{})
	if err != nil {
		return fmt.Errorf("update RoleBinding: %w", err)
	}

	a.emitRoleBindings()

	return nil
}

package app

import (
	"context"
	"fmt"
	"log"

	kubeResources "github.com/litelensapp/litelens/internal/kube/resources"
	"github.com/litelensapp/litelens/packages/core/kube/dto"
	"github.com/wailsapp/wails/v2/pkg/runtime"
	rbacv1 "k8s.io/api/rbac/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	sigsyaml "sigs.k8s.io/yaml"
)

func (a *App) GetRoleBindingByName(namespace, name string) (dto.RoleBinding, error) {
	h := a.activeFactory()
	if !waitForResourceSync(h, "rolebindings") {
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
	h, namespaces := a.activeFactoryAndNamespaces()
	if !waitForResourceSync(h, "rolebindings") {
		return []dto.RoleBinding{}, nil
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
	h, namespaces := a.activeFactoryAndNamespaces()
	if !waitForResourceSync(h, "rolebindings") {
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
	cs, err := a.activeClientset()
	if err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(context.Background(), apiMutationTimeout)
	defer cancel()
	err = cs.RbacV1().RoleBindings(namespace).Delete(ctx, name, metav1.DeleteOptions{})
	if err != nil && !errors.IsNotFound(err) {
		return fmt.Errorf("delete RoleBinding: %w", err)
	}

	a.emitRoleBindings()

	return nil
}

func (a *App) DeleteRoleBindings(items []dto.RoleBindingRef) error {
	cs, err := a.activeClientset()
	if err != nil {
		return err
	}

	err = deleteRefsBestEffort(items,
		func(r dto.RoleBindingRef) string { return r.Namespace },
		func(r dto.RoleBindingRef) string { return r.Name },
		"rolebindings",
		func(ctx context.Context, namespace, name string) error {
			return cs.RbacV1().RoleBindings(namespace).Delete(ctx, name, metav1.DeleteOptions{})
		},
	)

	a.emitRoleBindings()

	return err
}

func (a *App) GetRoleBindingYAML(namespace, name string) (string, error) {
	cs, err := a.activeClientset()
	if err != nil {
		return "", err
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
	cs, err := a.activeClientset()
	if err != nil {
		return err
	}

	var rb rbacv1.RoleBinding
	err = sigsyaml.Unmarshal([]byte(yamlString), &rb)
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

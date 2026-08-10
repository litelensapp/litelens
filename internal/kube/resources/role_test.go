package kubeResources

import (
	"errors"
	"testing"

	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	listersrbacv1 "k8s.io/client-go/listers/rbac/v1"
	"k8s.io/client-go/tools/cache"
)

type errorRoleLister struct{ err error }

func (e *errorRoleLister) List(_ labels.Selector) ([]*rbacv1.Role, error) {
	return nil, e.err
}
func (e *errorRoleLister) Roles(_ string) listersrbacv1.RoleNamespaceLister {
	return &errorRoleNamespaceLister{e.err}
}

type errorRoleNamespaceLister struct{ err error }

func (e *errorRoleNamespaceLister) List(_ labels.Selector) ([]*rbacv1.Role, error) {
	return nil, e.err
}
func (e *errorRoleNamespaceLister) Get(_ string) (*rbacv1.Role, error) {
	return nil, e.err
}

func newRoleLister(roles ...*rbacv1.Role) listersrbacv1.RoleLister {
	indexer := cache.NewIndexer(cache.MetaNamespaceKeyFunc, cache.Indexers{cache.NamespaceIndex: cache.MetaNamespaceIndexFunc})
	for _, r := range roles {
		_ = indexer.Add(r)
	}
	return listersrbacv1.NewRoleLister(indexer)
}

func makeRole(name, namespace string) *rbacv1.Role {
	return &rbacv1.Role{
		ObjectMeta: metav1.ObjectMeta{
			Name:              name,
			Namespace:         namespace,
			CreationTimestamp: metav1.Time{Time: fixedTime},
		},
	}
}

func TestListRoles_SingleNamespace(t *testing.T) {
	r := makeRole("reader", "default")
	lister := newRoleLister(r)

	result, err := ListRoles(lister, "default")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result) != 1 {
		t.Fatalf("expected 1 result, got %d", len(result))
	}
	if result[0].Name != "reader" {
		t.Errorf("Name = %q; want %q", result[0].Name, "reader")
	}
}

func TestListRoles_EmptyNamespaceReturnsAll(t *testing.T) {
	r1 := makeRole("r-a", "ns-a")
	r2 := makeRole("r-b", "ns-b")
	lister := newRoleLister(r1, r2)

	result, err := ListRoles(lister, "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result) != 2 {
		t.Errorf("expected 2 results, got %d", len(result))
	}
}

func TestListRoles_EmptyLister_ReturnsEmptySlice(t *testing.T) {
	lister := newRoleLister()

	result, err := ListRoles(lister, "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result == nil {
		t.Error("expected non-nil slice; got nil")
	}
	if len(result) != 0 {
		t.Errorf("expected empty result; got %d items", len(result))
	}
}

func TestListRoles_ErrorPropagation_ClusterScope(t *testing.T) {
	sentinel := errors.New("store unavailable")
	_, err := ListRoles(&errorRoleLister{err: sentinel}, "")
	if !errors.Is(err, sentinel) {
		t.Errorf("expected sentinel error; got %v", err)
	}
}

func TestListRoles_ErrorPropagation_NamespacedScope(t *testing.T) {
	sentinel := errors.New("namespace store unavailable")
	_, err := ListRoles(&errorRoleLister{err: sentinel}, "default")
	if !errors.Is(err, sentinel) {
		t.Errorf("expected sentinel error; got %v", err)
	}
}

func TestToRole_NilLabels_EmptyMap(t *testing.T) {
	r := makeRole("role", "default")
	r.Labels = nil

	got := toRole(r)
	if len(got.Labels) != 0 {
		t.Errorf("Labels length = %d; want 0", len(got.Labels))
	}
}

func TestToRole_NilAnnotations_EmptyMap(t *testing.T) {
	r := makeRole("role", "default")
	r.Annotations = nil

	got := toRole(r)
	if len(got.Annotations) != 0 {
		t.Errorf("Annotations length = %d; want 0", len(got.Annotations))
	}
}

func TestToRole_EmptyRules_EmptySlice(t *testing.T) {
	r := makeRole("role", "default")
	r.Rules = []rbacv1.PolicyRule{}

	got := toRole(r)
	if len(got.Rules) != 0 {
		t.Errorf("Rules length = %d; want 0", len(got.Rules))
	}
}

func TestToRole_WithRules(t *testing.T) {
	r := makeRole("role", "default")
	r.Rules = []rbacv1.PolicyRule{
		{
			APIGroups: []string{""},
			Resources: []string{"pods"},
			Verbs:     []string{"get", "list"},
		},
	}

	got := toRole(r)
	if len(got.Rules) != 1 {
		t.Fatalf("Rules length = %d; want 1", len(got.Rules))
	}
	if len(got.Rules[0].Resources) != 1 {
		t.Errorf("Rules[0].Resources length = %d; want 1", len(got.Rules[0].Resources))
	}
}

func TestToRole_NilResourcesSlice_EmptySlice(t *testing.T) {
	r := makeRole("role", "default")
	r.Rules = []rbacv1.PolicyRule{
		{
			APIGroups: []string{""},
			Resources: nil,
			Verbs:     []string{"create"},
		},
	}

	got := toRole(r)
	if len(got.Rules) != 1 {
		t.Fatalf("expected 1 rule")
	}
	if len(got.Rules[0].Resources) != 0 {
		t.Errorf("Resources length = %d; want 0", len(got.Rules[0].Resources))
	}
}

func TestGetRoleByName_Success(t *testing.T) {
	r := makeRole("reader", "default")
	r.Rules = []rbacv1.PolicyRule{
		{
			APIGroups: []string{""},
			Resources: []string{"pods"},
			Verbs:     []string{"get"},
		},
	}
	lister := newRoleLister(r)

	result, err := GetRoleByName(lister, "default", "reader")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Name != "reader" {
		t.Errorf("Name = %q; want %q", result.Name, "reader")
	}
}

func TestGetRoleByName_NotFound(t *testing.T) {
	r := makeRole("reader", "default")
	lister := newRoleLister(r)

	_, err := GetRoleByName(lister, "default", "nonexistent")
	if err == nil {
		t.Error("expected error for nonexistent Role; got nil")
	}
}

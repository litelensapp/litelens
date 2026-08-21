package kubeResources

import (
	"testing"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	listersappsv1 "k8s.io/client-go/listers/apps/v1"
	"k8s.io/client-go/tools/cache"
)

func newStatefulSetLister(sss ...*appsv1.StatefulSet) listersappsv1.StatefulSetLister {
	indexer := cache.NewIndexer(cache.MetaNamespaceKeyFunc, cache.Indexers{cache.NamespaceIndex: cache.MetaNamespaceIndexFunc})
	for _, ss := range sss {
		_ = indexer.Add(ss)
	}
	return listersappsv1.NewStatefulSetLister(indexer)
}

func makeStatefulSet(name, namespace string) *appsv1.StatefulSet {
	return &appsv1.StatefulSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:              name,
			Namespace:         namespace,
			CreationTimestamp: metav1.Time{Time: fixedTime},
		},
		Status: appsv1.StatefulSetStatus{
			ReadyReplicas:     1,
			AvailableReplicas: 1,
		},
	}
}

func TestListStatefulSets_SingleNamespace(t *testing.T) {
	ss := makeStatefulSet("db", "default")
	lister := newStatefulSetLister(ss)

	result, err := ListStatefulSets(lister, []string{"default"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result) != 1 {
		t.Fatalf("expected 1 result, got %d", len(result))
	}
	if result[0].Name != "db" {
		t.Errorf("Name = %q; want %q", result[0].Name, "db")
	}
}

func TestListStatefulSets_EmptyNamespaceReturnsAll(t *testing.T) {
	ss1 := makeStatefulSet("ss-a", "ns-a")
	ss2 := makeStatefulSet("ss-b", "ns-b")
	lister := newStatefulSetLister(ss1, ss2)

	result, err := ListStatefulSets(lister, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result) != 2 {
		t.Errorf("expected 2 results, got %d", len(result))
	}
}

func TestListStatefulSets_EmptyLister_ReturnsEmptySlice(t *testing.T) {
	lister := newStatefulSetLister()

	result, err := ListStatefulSets(lister, nil)
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

func TestToStatefulSet_NilReplicas_DefaultsToOne(t *testing.T) {
	ss := makeStatefulSet("ss", "default")
	ss.Spec.Replicas = nil

	got := toStatefulSet(ss)
	if got.Replicas != 1 {
		t.Errorf("Replicas = %d; want 1", got.Replicas)
	}
}

func TestToStatefulSet_WithReplicas(t *testing.T) {
	ss := makeStatefulSet("ss", "default")
	replicas := int32(3)
	ss.Spec.Replicas = &replicas

	got := toStatefulSet(ss)
	if got.Replicas != 3 {
		t.Errorf("Replicas = %d; want 3", got.Replicas)
	}
}

func TestToStatefulSet_PodStatus(t *testing.T) {
	ss := makeStatefulSet("ss", "default")
	replicas := int32(5)
	ss.Spec.Replicas = &replicas
	ss.Status.ReadyReplicas = 4
	ss.Status.AvailableReplicas = 3

	got := toStatefulSet(ss)
	if !contains(got.PodStatus, "5 desired") {
		t.Errorf("PodStatus = %q; should contain '5 desired'", got.PodStatus)
	}
	if !contains(got.PodStatus, "4 ready") {
		t.Errorf("PodStatus = %q; should contain '4 ready'", got.PodStatus)
	}
	if !contains(got.PodStatus, "3 available") {
		t.Errorf("PodStatus = %q; should contain '3 available'", got.PodStatus)
	}
}

func TestToStatefulSet_NilSelector_EmptySelector(t *testing.T) {
	ss := makeStatefulSet("ss", "default")
	ss.Spec.Selector = nil

	got := toStatefulSet(ss)
	if got.Selector != "" {
		t.Errorf("Selector = %q; want empty string", got.Selector)
	}
}

func TestToStatefulSet_WithSelector(t *testing.T) {
	ss := makeStatefulSet("ss", "default")
	ss.Spec.Selector = &metav1.LabelSelector{
		MatchLabels: map[string]string{"app": "mysql"},
	}

	got := toStatefulSet(ss)
	if got.Selector != "app=mysql" {
		t.Errorf("Selector = %q; want %q", got.Selector, "app=mysql")
	}
}

func TestToStatefulSet_NilLabels_EmptyMap(t *testing.T) {
	ss := makeStatefulSet("ss", "default")
	ss.Labels = nil

	got := toStatefulSet(ss)
	if len(got.Labels) != 0 {
		t.Errorf("Labels length = %d; want 0", len(got.Labels))
	}
}

func TestToStatefulSet_WithAffinity(t *testing.T) {
	ss := makeStatefulSet("ss", "default")
	ss.Spec.Template.Spec.Affinity = &corev1.Affinity{}

	got := toStatefulSet(ss)
	if got.Affinities != 1 {
		t.Errorf("Affinities = %d; want 1", got.Affinities)
	}
}

func TestGetStatefulSetByName_Success(t *testing.T) {
	ss := makeStatefulSet("db", "default")
	lister := newStatefulSetLister(ss)

	result, err := GetStatefulSetByName(lister, "default", "db")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Name != "db" {
		t.Errorf("Name = %q; want %q", result.Name, "db")
	}
}

func TestGetStatefulSetByName_NotFound(t *testing.T) {
	ss := makeStatefulSet("db", "default")
	lister := newStatefulSetLister(ss)

	_, err := GetStatefulSetByName(lister, "default", "nonexistent")
	if err == nil {
		t.Error("expected error for nonexistent StatefulSet; got nil")
	}
}

func contains(s, substr string) bool {
	for i := 0; i+len(substr) <= len(s); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

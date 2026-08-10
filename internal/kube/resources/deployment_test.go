package kubeResources

import (
	"strings"
	"testing"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	listersappsv1 "k8s.io/client-go/listers/apps/v1"
	"k8s.io/client-go/tools/cache"
)

type errorDeploymentLister struct{ err error }

func (e *errorDeploymentLister) List(_ labels.Selector) ([]*appsv1.Deployment, error) {
	return nil, e.err
}
func (e *errorDeploymentLister) Deployments(_ string) listersappsv1.DeploymentNamespaceLister {
	return &errorDeploymentNamespaceLister{e.err}
}

type errorDeploymentNamespaceLister struct{ err error }

func (e *errorDeploymentNamespaceLister) List(_ labels.Selector) ([]*appsv1.Deployment, error) {
	return nil, e.err
}
func (e *errorDeploymentNamespaceLister) Get(_ string) (*appsv1.Deployment, error) {
	return nil, e.err
}

func newDeploymentLister(deps ...*appsv1.Deployment) listersappsv1.DeploymentLister {
	indexer := cache.NewIndexer(cache.MetaNamespaceKeyFunc, cache.Indexers{cache.NamespaceIndex: cache.MetaNamespaceIndexFunc})
	for _, d := range deps {
		_ = indexer.Add(d)
	}
	return listersappsv1.NewDeploymentLister(indexer)
}

func makeDeployment(name, namespace string) *appsv1.Deployment {
	return &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:              name,
			Namespace:         namespace,
			CreationTimestamp: metav1.Time{Time: fixedTime},
		},
		Spec:   appsv1.DeploymentSpec{},
		Status: appsv1.DeploymentStatus{},
	}
}

func TestToDeployment_NodeSelectorNil_ReturnsEmptyString(t *testing.T) {
	d := makeDeployment("d1", "default")
	d.Spec.Template.Spec.NodeSelector = nil
	got := toDeployment(d)
	if got.NodeSelector != "" {
		t.Errorf("NodeSelector = %q; want empty string", got.NodeSelector)
	}
}

func TestToDeployment_NodeSelectorEmpty_ReturnsEmptyString(t *testing.T) {
	d := makeDeployment("d1", "default")
	d.Spec.Template.Spec.NodeSelector = map[string]string{}
	got := toDeployment(d)
	if got.NodeSelector != "" {
		t.Errorf("NodeSelector = %q; want empty string", got.NodeSelector)
	}
}

func TestToDeployment_NodeSelectorMultiEntry_SortedByKey(t *testing.T) {
	d := makeDeployment("d1", "default")
	d.Spec.Template.Spec.NodeSelector = map[string]string{
		"zone":   "us-west",
		"disk":   "ssd",
		"memory": "high",
	}
	got := toDeployment(d)
	parts := strings.Split(got.NodeSelector, ", ")
	if len(parts) != 3 {
		t.Fatalf("expected 3 parts; got %d: %q", len(parts), got.NodeSelector)
	}
	if parts[0] != "disk=ssd" || parts[1] != "memory=high" || parts[2] != "zone=us-west" {
		t.Errorf("NodeSelector not sorted correctly: %q", got.NodeSelector)
	}
}

func TestToDeployment_ConditionsEmpty_ReturnsEmptySlice(t *testing.T) {
	d := makeDeployment("d1", "default")
	d.Status.Conditions = []appsv1.DeploymentCondition{}
	got := toDeployment(d)
	if got.Conditions == nil {
		t.Error("Conditions must not be nil for empty input slice")
	}
	if len(got.Conditions) != 0 {
		t.Errorf("Conditions length = %d; want 0", len(got.Conditions))
	}
}

func TestToDeployment_TolerationsEmpty_ReturnsZeroAndEmptySlice(t *testing.T) {
	d := makeDeployment("d1", "default")
	d.Spec.Template.Spec.Tolerations = []corev1.Toleration{}
	got := toDeployment(d)
	if got.Tolerations != 0 {
		t.Errorf("Tolerations count = %d; want 0", got.Tolerations)
	}
	if got.TolerationDetails == nil {
		t.Error("TolerationDetails must not be nil")
	}
	if len(got.TolerationDetails) != 0 {
		t.Errorf("TolerationDetails length = %d; want 0", len(got.TolerationDetails))
	}
}

func TestToDeployment_AffinityNil_CountZeroAndEmptyString(t *testing.T) {
	d := makeDeployment("d1", "default")
	d.Spec.Template.Spec.Affinity = nil
	got := toDeployment(d)
	if got.AffinityCount != 0 {
		t.Errorf("AffinityCount = %d; want 0", got.AffinityCount)
	}
	if got.Affinities != "" {
		t.Errorf("Affinities = %q; want empty string", got.Affinities)
	}
}

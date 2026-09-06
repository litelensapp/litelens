package nsscope

import (
	"fmt"
	"testing"

	appsv1 "k8s.io/api/apps/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes/fake"
	"k8s.io/client-go/tools/cache"
)

func newDeployment(namespace, name string) *appsv1.Deployment {
	return &appsv1.Deployment{ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: namespace}}
}

func TestNewDeploymentsResourceNamespaceScoped(t *testing.T) {
	objs := []runtime.Object{
		newDeployment("ns-a", "a1"), newDeployment("ns-a", "a2"),
		newDeployment("ns-b", "b1"),
		newDeployment("ns-c", "c1"), // deliberately not selected
	}
	cs := fake.NewSimpleClientset(objs...)
	factory := informers.NewSharedInformerFactory(cs, 0)
	r := NewDeploymentsResource(cs, func() cache.SharedIndexInformer { return factory.Apps().V1().Deployments().Informer() }, nil, nil)
	defer r.Stop()

	r.Rescope([]string{"ns-a", "ns-b"})
	<-r.SyncedChan()

	all, err := r.Lister().List(labels.Everything())
	if err != nil {
		t.Fatalf("unexpected error listing deployments: %v", err)
	}
	if len(all) != 3 {
		t.Fatalf("expected 3 deployments across ns-a/ns-b, got %d", len(all))
	}
	for _, d := range all {
		if d.Namespace == "ns-c" {
			t.Fatalf("expected no deployments from unselected namespace ns-c, got %v", d)
		}
	}
}

func TestNewDeploymentsResourceFallsBackToClusterWideBeyondThreshold(t *testing.T) {
	var objs []runtime.Object
	var namespaces []string
	for i := range maxNamespaceScopedDeploymentInformers + 1 {
		ns := fmt.Sprintf("ns-%d", i)
		namespaces = append(namespaces, ns)
		objs = append(objs, newDeployment(ns, "dep"))
	}
	objs = append(objs, newDeployment("ns-other", "dep"))

	cs := fake.NewSimpleClientset(objs...)
	factory := informers.NewSharedInformerFactory(cs, 0)
	r := NewDeploymentsResource(cs, func() cache.SharedIndexInformer { return factory.Apps().V1().Deployments().Informer() }, nil, nil)
	defer r.Stop()

	r.Rescope(namespaces)
	<-r.SyncedChan()

	all, err := r.Lister().List(labels.Everything())
	if err != nil {
		t.Fatalf("unexpected error listing deployments: %v", err)
	}
	if len(all) != len(namespaces)+1 {
		t.Fatalf("expected cluster-wide fallback to see all %d deployments, got %d", len(namespaces)+1, len(all))
	}
}

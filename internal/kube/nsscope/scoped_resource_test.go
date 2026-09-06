package nsscope

import (
	"fmt"
	"testing"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	corev1informers "k8s.io/client-go/informers/core/v1"
	"k8s.io/client-go/kubernetes/fake"
	listerscorev1 "k8s.io/client-go/listers/core/v1"
	"k8s.io/client-go/tools/cache"
)

// The tests below exercise the generic engine using Pods as a concrete,
// realistic instantiation — resource-specific glue lives in the parent kube
// package's pods.go, but the rebuild/swap/fallback logic under test here is
// entirely generic.

func newPod(namespace, name string) *corev1.Pod {
	return &corev1.Pod{ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: namespace}}
}

type multiPodLister struct {
	listers map[string]listerscorev1.PodLister
}

func (m *multiPodLister) List(selector labels.Selector) ([]*corev1.Pod, error) {
	var all []*corev1.Pod
	for _, l := range m.listers {
		pods, err := l.List(selector)
		if err != nil {
			return nil, err
		}
		all = append(all, pods...)
	}
	return all, nil
}

func (m *multiPodLister) Pods(namespace string) listerscorev1.PodNamespaceLister {
	return m.listers[namespace].Pods(namespace)
}

func newTestScopedResource(cs *fake.Clientset) *ScopedResource[listerscorev1.PodLister] {
	return New(Config[listerscorev1.PodLister]{
		Name:          "pods",
		MaxNamespaces: 10,
		NewNamespacedInformer: func(ns string) cache.SharedIndexInformer {
			indexers := cache.Indexers{cache.NamespaceIndex: cache.MetaNamespaceIndexFunc}
			return corev1informers.NewFilteredPodInformer(cs, ns, 0, indexers, nil)
		},
		NewClusterWideInformer: func() cache.SharedIndexInformer {
			indexers := cache.Indexers{cache.NamespaceIndex: cache.MetaNamespaceIndexFunc}
			return corev1informers.NewFilteredPodInformer(cs, "", 0, indexers, nil)
		},
		BuildLister: func(indexer cache.Indexer) listerscorev1.PodLister {
			return listerscorev1.NewPodLister(indexer)
		},
		BuildMultiLister: func(m map[string]listerscorev1.PodLister) listerscorev1.PodLister {
			return &multiPodLister{listers: m}
		},
	})
}

func TestRescopeNamespaceScoped(t *testing.T) {
	objs := []runtime.Object{
		newPod("ns-a", "a1"), newPod("ns-a", "a2"),
		newPod("ns-b", "b1"), newPod("ns-b", "b2"), newPod("ns-b", "b3"),
		newPod("ns-c", "c1"), // deliberately not selected
	}
	cs := fake.NewSimpleClientset(objs...)
	r := newTestScopedResource(cs)
	defer r.Stop()

	r.Rescope([]string{"ns-a", "ns-b"})
	<-r.SyncedChan()

	all, err := r.Lister().List(labels.Everything())
	if err != nil {
		t.Fatalf("unexpected error listing pods: %v", err)
	}
	if len(all) != 5 {
		t.Fatalf("expected 5 pods across ns-a/ns-b, got %d", len(all))
	}
	for _, p := range all {
		if p.Namespace == "ns-c" {
			t.Fatalf("expected no pods from unselected namespace ns-c, got %v", p)
		}
	}
}

func TestRescopeFallsBackToClusterWideBeyondThreshold(t *testing.T) {
	var objs []runtime.Object
	var namespaces []string
	for i := range 11 {
		ns := fmt.Sprintf("ns-%d", i)
		namespaces = append(namespaces, ns)
		objs = append(objs, newPod(ns, "pod"))
	}
	objs = append(objs, newPod("ns-other", "pod"))

	cs := fake.NewSimpleClientset(objs...)
	r := newTestScopedResource(cs)
	defer r.Stop()

	r.Rescope(namespaces)
	<-r.SyncedChan()

	all, err := r.Lister().List(labels.Everything())
	if err != nil {
		t.Fatalf("unexpected error listing pods: %v", err)
	}
	if len(all) != len(namespaces)+1 {
		t.Fatalf("expected cluster-wide fallback to see all %d pods, got %d", len(namespaces)+1, len(all))
	}
}

func TestRescopeNoOpWhenUnchanged(t *testing.T) {
	cs := fake.NewSimpleClientset()
	r := newTestScopedResource(cs)
	defer r.Stop()

	r.Rescope([]string{"ns-a", "ns-b"})
	<-r.SyncedChan()
	first := r.group

	r.Rescope([]string{"ns-b", "ns-a"})
	if r.group != first {
		t.Fatal("expected Rescope to no-op for an equivalent (reordered) namespace set")
	}
}

func TestRescopeSwapStopsOldGroup(t *testing.T) {
	cs := fake.NewSimpleClientset()
	r := newTestScopedResource(cs)
	defer r.Stop()

	r.Rescope([]string{"ns-a"})
	<-r.SyncedChan()
	old := r.group

	r.Rescope([]string{"ns-a", "ns-b"})
	<-r.SyncedChan()

	select {
	case <-old.stop:
		// Expected: superseded group's informers were stopped.
	default:
		t.Fatal("expected old group's stop channel to be closed after rescope")
	}
}

func TestMarkForbiddenIgnoresSupersededGroup(t *testing.T) {
	cs := fake.NewSimpleClientset()
	var forbiddenCalls []string
	r := New(Config[listerscorev1.PodLister]{
		Name:          "pods",
		MaxNamespaces: 10,
		NewNamespacedInformer: func(ns string) cache.SharedIndexInformer {
			indexers := cache.Indexers{cache.NamespaceIndex: cache.MetaNamespaceIndexFunc}
			return corev1informers.NewFilteredPodInformer(cs, ns, 0, indexers, nil)
		},
		NewClusterWideInformer: func() cache.SharedIndexInformer {
			indexers := cache.Indexers{cache.NamespaceIndex: cache.MetaNamespaceIndexFunc}
			return corev1informers.NewFilteredPodInformer(cs, "", 0, indexers, nil)
		},
		BuildLister: func(indexer cache.Indexer) listerscorev1.PodLister {
			return listerscorev1.NewPodLister(indexer)
		},
		BuildMultiLister: func(m map[string]listerscorev1.PodLister) listerscorev1.PodLister {
			return &multiPodLister{listers: m}
		},
		OnForbidden: func(name string) { forbiddenCalls = append(forbiddenCalls, name) },
	})
	defer r.Stop()

	r.Rescope([]string{"ns-a"})
	<-r.SyncedChan()
	old := r.group

	r.Rescope([]string{"ns-b"})
	<-r.SyncedChan()

	// Simulate the old (superseded, already-stopped) group's sync-wait
	// discovering a failure after the swap — it must not poison the new
	// group's forbidden state.
	r.markForbidden(old)

	if len(forbiddenCalls) != 0 {
		t.Fatalf("expected superseded group's failure to be ignored, got OnForbidden calls: %v", forbiddenCalls)
	}
}

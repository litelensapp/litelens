package app

import (
	"fmt"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/litelensapp/litelens/internal/kube"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes/fake"
)

// TestGetSyncedChanNilHandle verifies that GetSyncedChan on nil *FactoryHandle
// returns an already-closed channel (no panic, immediate return).
func TestGetSyncedChanNilHandle(t *testing.T) {
	var h *kube.FactoryHandle
	ch := h.GetSyncedChan("pods")

	select {
	case <-ch:
		// Expected: immediately returns due to already-closed channel
	case <-time.After(100 * time.Millisecond):
		t.Fatal("GetSyncedChan on nil handle should return immediately")
	}
}

// TestGetSyncedChanConcurrentReads verifies that multiple goroutines can
// concurrently call GetSyncedChan for the same resource and all unblock
// when the resource's cache sync completes.
func TestGetSyncedChanConcurrentReads(t *testing.T) {
	const podCount = 100
	const readerCount = 10

	objs := make([]runtime.Object, 0, podCount)
	for i := range podCount {
		objs = append(objs, &corev1.Pod{
			ObjectMeta: metav1.ObjectMeta{
				Name:      fmt.Sprintf("pod-%d", i),
				Namespace: "default",
			},
		})
	}

	cs := fake.NewSimpleClientset(objs...)
	h := kube.NewFactoryHandle(cs, func(string) {})
	defer h.Stop()

	// Spawn multiple concurrent readers on the same resource
	var wg sync.WaitGroup
	var unblockCount atomic.Int32
	var readyMutex sync.Mutex
	readyCount := 0

	for r := range readerCount {
		wg.Add(1)
		go func(readerID int) {
			defer wg.Done()
			ch := h.GetSyncedChan("pods")
			readyMutex.Lock()
			readyCount++
			readyMutex.Unlock()

			// Block until cache sync completes
			<-ch
			unblockCount.Add(1)
		}(r)
	}

	// Wait a bit to ensure all readers are waiting on the channel
	time.Sleep(100 * time.Millisecond)

	wg.Wait()

	if unblockCount.Load() != int32(readerCount) {
		t.Fatalf("expected all %d readers to unblock, got %d", readerCount, unblockCount.Load())
	}
}

// TestGetSyncedChanAfterForbidden verifies that after a resource is marked
// forbidden via stopResource, GetSyncedChan still returns a closed channel,
// and IsForbidden reflects the state change.
func TestGetSyncedChanAfterForbidden(t *testing.T) {
	cs := fake.NewSimpleClientset()
	h := kube.NewFactoryHandle(cs, func(string) {})
	defer h.Stop()

	// Initially, resource should not be forbidden
	if h.IsForbidden("pods") {
		t.Fatal("pods should not be forbidden initially")
	}

	// Mark the resource as forbidden
	h.StopResource("pods", func(string) {})

	// After forbidden, IsForbidden should return true
	if !h.IsForbidden("pods") {
		t.Fatal("pods should be forbidden after stopResource")
	}

	// GetSyncedChan should still return a closed channel (no panic)
	ch := h.GetSyncedChan("pods")
	select {
	case <-ch:
		// Expected: channel is closed
	case <-time.After(100 * time.Millisecond):
		t.Fatal("GetSyncedChan should return an open or already-closed channel")
	}
}

// TestListConfigMapsGatingPattern verifies that ListConfigMaps correctly gates
// on GetSyncedChan("configmaps") and short-circuits on IsForbidden.
func TestListConfigMapsGatingPattern(t *testing.T) {
	const cmCount = 5

	objs := make([]runtime.Object, 0, cmCount)
	for i := range cmCount {
		objs = append(objs, &corev1.ConfigMap{
			ObjectMeta: metav1.ObjectMeta{
				Name:      fmt.Sprintf("cm-%d", i),
				Namespace: "default",
			},
		})
	}

	cs := fake.NewSimpleClientset(objs...)
	h := kube.NewFactoryHandle(cs, func(string) {})
	defer h.Stop()

	// Create an App with this factory handle
	a := &App{
		activeContext: "test",
		factories: map[string]*kube.FactoryHandle{
			"test": h,
		},
	}

	// Wait for cache sync
	<-h.GetSyncedChan("configmaps")

	// ListConfigMaps should return the ConfigMaps
	cms, err := a.ListConfigMaps("default")
	if err != nil {
		t.Fatalf("ListConfigMaps failed: %v", err)
	}
	if len(cms) != cmCount {
		t.Fatalf("expected %d ConfigMaps, got %d", cmCount, len(cms))
	}

	// Now mark configmaps as forbidden
	h.StopResource("configmaps", func(string) {})

	// ListConfigMaps should return an empty slice (zero-value)
	cms2, err := a.ListConfigMaps("default")
	if err != nil {
		t.Fatalf("ListConfigMaps with forbidden resource should return nil error, got %v", err)
	}
	if len(cms2) != 0 {
		t.Fatalf("expected empty ConfigMaps slice after forbidden, got %d", len(cms2))
	}
}

// TestGetConfigMapByNameGatingPattern verifies that GetConfigMapByName correctly
// gates on GetSyncedChan("configmaps") and short-circuits on IsForbidden.
func TestGetConfigMapByNameGatingPattern(t *testing.T) {
	objs := []runtime.Object{
		&corev1.ConfigMap{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-cm",
				Namespace: "default",
			},
			Data: map[string]string{"key": "value"},
		},
	}

	cs := fake.NewSimpleClientset(objs...)
	h := kube.NewFactoryHandle(cs, func(string) {})
	defer h.Stop()

	a := &App{
		activeContext: "test",
		factories: map[string]*kube.FactoryHandle{
			"test": h,
		},
	}

	// Wait for cache sync
	<-h.GetSyncedChan("configmaps")

	// GetConfigMapByName should return the ConfigMap
	cm, err := a.GetConfigMapByName("default", "test-cm")
	if err != nil {
		t.Fatalf("GetConfigMapByName failed: %v", err)
	}
	if cm.Name != "test-cm" {
		t.Fatalf("expected ConfigMap name 'test-cm', got %q", cm.Name)
	}

	// Mark configmaps as forbidden
	h.StopResource("configmaps", func(string) {})

	// GetConfigMapByName should return zero-value with nil error
	cm2, err := a.GetConfigMapByName("default", "test-cm")
	if err != nil {
		t.Fatalf("GetConfigMapByName with forbidden resource should return nil error, got %v", err)
	}
	if cm2.Name != "" {
		t.Fatalf("expected zero-value ConfigMap after forbidden, got %q", cm2.Name)
	}
}

// TestConcurrentGetSyncedChanWithStopResource verifies that concurrent calls to
// GetSyncedChan for multiple resources while stopResource is being called
// do not cause races or double-closes.
func TestConcurrentGetSyncedChanWithStopResource(t *testing.T) {
	cs := fake.NewSimpleClientset()
	h := kube.NewFactoryHandle(cs, func(string) {})
	defer h.Stop()

	const goroutineCount = 20
	resources := []string{"pods", "configmaps", "deployments", "services"}

	var wg sync.WaitGroup

	// Spawn readers for all resources
	for i := range goroutineCount {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			resource := resources[id%len(resources)]
			ch := h.GetSyncedChan(resource)
			// Should not panic, even if stopResource is called concurrently
			<-ch
		}(i)
	}

	// Spawn concurrent stopResource calls
	for i := range goroutineCount {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			resource := resources[id%len(resources)]
			h.StopResource(resource, func(string) {})
		}(i)
	}

	// Wait for all goroutines
	wg.Wait()

	// Verify that no panics occurred and all resources are marked forbidden
	for _, resource := range resources {
		if !h.IsForbidden(resource) {
			t.Fatalf("expected %s to be forbidden after concurrent stopResource calls", resource)
		}
	}
}

// TestUnknownResourceIsForbiddenFalse verifies that querying IsForbidden for
// a resource that was never registered returns false.
func TestUnknownResourceIsForbiddenFalse(t *testing.T) {
	cs := fake.NewSimpleClientset()
	h := kube.NewFactoryHandle(cs, func(string) {})
	defer h.Stop()

	if h.IsForbidden("unknown-resource") {
		t.Fatal("expected IsForbidden to return false for unknown resource")
	}
}

// TestGetSyncedChanUnknownResourceImmediate verifies that calling GetSyncedChan
// for an unknown resource returns an already-closed channel (immediate return).
func TestGetSyncedChanUnknownResourceImmediate(t *testing.T) {
	cs := fake.NewSimpleClientset()
	h := kube.NewFactoryHandle(cs, func(string) {})
	defer h.Stop()

	ch := h.GetSyncedChan("unknown-resource")

	select {
	case <-ch:
		// Expected: channel is closed
	case <-time.After(100 * time.Millisecond):
		t.Fatal("GetSyncedChan for unknown resource should return immediately")
	}
}

// TestListPodsAfterCacheSyncGates verifies that ListPods correctly gates on
// the sync channel before returning results.
func TestListPodsAfterCacheSyncGates(t *testing.T) {
	objs := []runtime.Object{
		&corev1.Pod{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-pod",
				Namespace: "default",
			},
		},
	}

	cs := fake.NewSimpleClientset(objs...)
	h := kube.NewFactoryHandle(cs, func(string) {})
	defer h.Stop()

	a := &App{
		activeContext: "test",
		factories: map[string]*kube.FactoryHandle{
			"test": h,
		},
	}

	// Wait for cache sync
	<-h.GetSyncedChan("pods")

	// ListPods should return the pod
	pods, err := a.ListPods("default")
	if err != nil {
		t.Fatalf("ListPods failed: %v", err)
	}
	if len(pods) != 1 || pods[0].Name != "test-pod" {
		t.Fatalf("expected 1 pod named 'test-pod', got %v", pods)
	}
}

// TestIsForbiddenBecomesTrueAfterStopResource verifies that IsForbidden changes
// state from false to true after stopResource is called.
func TestIsForbiddenBecomesTrueAfterStopResource(t *testing.T) {
	cs := fake.NewSimpleClientset()
	h := kube.NewFactoryHandle(cs, func(string) {})
	defer h.Stop()

	resource := "pods"

	// Before stopResource, should be false
	if h.IsForbidden(resource) {
		t.Fatalf("expected IsForbidden(%q) to be false before stopResource", resource)
	}

	// Call stopResource
	h.StopResource(resource, func(string) {})

	// After stopResource, should be true
	if !h.IsForbidden(resource) {
		t.Fatalf("expected IsForbidden(%q) to be true after stopResource", resource)
	}
}

// TestMultipleStopResourceCallsIdempotent verifies that calling stopResource
// multiple times for the same resource doesn't panic or cause issues
// (sync.Once guard in the implementation).
func TestMultipleStopResourceCallsIdempotent(t *testing.T) {
	cs := fake.NewSimpleClientset()
	h := kube.NewFactoryHandle(cs, func(string) {})
	defer h.Stop()

	resource := "pods"
	callCount := 0
	onForbidden := func(string) {
		callCount++
	}

	// Call stopResource multiple times
	h.StopResource(resource, onForbidden)
	h.StopResource(resource, onForbidden)
	h.StopResource(resource, onForbidden)

	// onForbidden may have been called multiple times (no dedup), but no panic
	if !h.IsForbidden(resource) {
		t.Fatal("expected resource to be forbidden")
	}
}

// TestForbiddenResourceReturnsZeroValueAndNilError verifies that App methods
// return zero-value and nil error when the resource is forbidden.
func TestForbiddenResourceReturnsZeroValueAndNilError(t *testing.T) {
	cs := fake.NewSimpleClientset()
	h := kube.NewFactoryHandle(cs, func(string) {})
	defer h.Stop()

	a := &App{
		activeContext: "test",
		factories: map[string]*kube.FactoryHandle{
			"test": h,
		},
	}

	// Mark pods as forbidden
	h.StopResource("pods", func(string) {})

	// ListPods should return empty slice and nil error
	pods, err := a.ListPods("default")
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	if len(pods) != 0 {
		t.Fatalf("expected empty slice for forbidden resource, got %d pods", len(pods))
	}

	// GetPodByName should return zero-value and nil error
	pod, err := a.GetPodByName("default", "nonexistent")
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	if pod.Name != "" {
		t.Fatalf("expected zero-value pod, got %q", pod.Name)
	}
}

// TestGetNodeByNameGatingPattern verifies that GetNodeByName correctly gates
// on GetSyncedChan("nodes") and short-circuits on IsForbidden. This covers
// cluster-scoped resources (nodes, unlike namespaced ConfigMaps).
func TestGetNodeByNameGatingPattern(t *testing.T) {
	objs := []runtime.Object{
		&corev1.Node{
			ObjectMeta: metav1.ObjectMeta{
				Name: "node-1",
			},
			Status: corev1.NodeStatus{},
		},
	}

	cs := fake.NewSimpleClientset(objs...)
	h := kube.NewFactoryHandle(cs, func(string) {})
	defer h.Stop()

	a := &App{
		activeContext: "test",
		factories: map[string]*kube.FactoryHandle{
			"test": h,
		},
	}

	// Wait for cache sync
	<-h.GetSyncedChan("nodes")

	// GetNodeByName should return the node
	node, err := a.GetNodeByName("node-1")
	if err != nil {
		t.Fatalf("GetNodeByName failed: %v", err)
	}
	if node.Name != "node-1" {
		t.Fatalf("expected node name 'node-1', got %q", node.Name)
	}

	// Mark nodes as forbidden
	h.StopResource("nodes", func(string) {})

	// GetNodeByName should return zero-value with nil error
	node2, err := a.GetNodeByName("node-1")
	if err != nil {
		t.Fatalf("GetNodeByName with forbidden resource should return nil error, got %v", err)
	}
	if node2.Name != "" {
		t.Fatalf("expected zero-value node after forbidden, got %q", node2.Name)
	}
}

// TestListNodesGatingPattern verifies that ListNodes correctly gates on
// GetSyncedChan("nodes") and short-circuits on IsForbidden.
func TestListNodesGatingPattern(t *testing.T) {
	objs := []runtime.Object{
		&corev1.Node{
			ObjectMeta: metav1.ObjectMeta{
				Name: "node-1",
			},
		},
		&corev1.Node{
			ObjectMeta: metav1.ObjectMeta{
				Name: "node-2",
			},
		},
	}

	cs := fake.NewSimpleClientset(objs...)
	h := kube.NewFactoryHandle(cs, func(string) {})
	defer h.Stop()

	a := &App{
		activeContext: "test",
		factories: map[string]*kube.FactoryHandle{
			"test": h,
		},
	}

	// Wait for cache sync
	<-h.GetSyncedChan("nodes")

	// ListNodes should return the nodes
	nodes, err := a.ListNodes()
	if err != nil {
		t.Fatalf("ListNodes failed: %v", err)
	}
	if len(nodes) != 2 {
		t.Fatalf("expected 2 nodes, got %d", len(nodes))
	}

	// Mark nodes as forbidden
	h.StopResource("nodes", func(string) {})

	// ListNodes should return an empty slice (zero-value)
	nodes2, err := a.ListNodes()
	if err != nil {
		t.Fatalf("ListNodes with forbidden resource should return nil error, got %v", err)
	}
	if len(nodes2) != 0 {
		t.Fatalf("expected empty nodes slice after forbidden, got %d", len(nodes2))
	}
}

// TestGetSyncedChanMultipleResourcesInSequence verifies that GetSyncedChan
// returns distinct channels for different resources and all close properly.
func TestGetSyncedChanMultipleResourcesInSequence(t *testing.T) {
	cs := fake.NewSimpleClientset()
	h := kube.NewFactoryHandle(cs, func(string) {})
	defer h.Stop()

	resources := []string{"pods", "configmaps", "deployments"}
	channels := make(map[string]<-chan struct{})

	for _, resource := range resources {
		channels[resource] = h.GetSyncedChan(resource)
	}

	// All channels should close (real resources) or already be closed (due to async sync)
	for _, resource := range resources {
		select {
		case <-channels[resource]:
			// Expected: channel is closed
		case <-time.After(1 * time.Second):
			t.Fatalf("GetSyncedChan(%q) did not close within timeout", resource)
		}
	}
}

// BenchmarkGetSyncedChan benchmarks the cost of calling GetSyncedChan.
func BenchmarkGetSyncedChan(b *testing.B) {
	cs := fake.NewSimpleClientset()
	h := kube.NewFactoryHandle(cs, func(string) {})
	defer h.Stop()

	b.ResetTimer()
	for range b.N {
		h.GetSyncedChan("pods")
	}
}

// BenchmarkIsForbidden benchmarks the cost of checking if a resource is forbidden.
func BenchmarkIsForbidden(b *testing.B) {
	cs := fake.NewSimpleClientset()
	h := kube.NewFactoryHandle(cs, func(string) {})
	defer h.Stop()

	b.ResetTimer()
	for range b.N {
		h.IsForbidden("pods")
	}
}

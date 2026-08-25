package grpc

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/litelensapp/litelens/packages/core/pb"
	"google.golang.org/grpc/metadata"
)

// TestHostPluginServer_SubscriptionReplay tests that new subscribers receive the last published event immediately.
func TestHostPluginServer_SubscriptionReplay(t *testing.T) {
	server := NewHostPluginServer(func(payload map[string]interface{}) {})
	defer server.MarkStopped()

	// Publish an event before any subscriptions
	server.PublishClusterContextChange("initial-context", "/path/to/initial/kubeconfig")

	// Create a subscriber with a timeout context
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	mockStream := &mockStream{
		ctx:   ctx,
		recvd: make([]*pb.ClusterContextChangedEvent, 0),
	}

	// Subscribe to watch
	server.ClusterContextWatch(&pb.Empty{}, mockStream)

	// Verify the initial event was replayed
	if len(mockStream.recvd) != 1 {
		t.Fatalf("expected 1 event from replay, got %d", len(mockStream.recvd))
	}
	if mockStream.recvd[0].ContextName != "initial-context" {
		t.Fatalf("expected context name 'initial-context', got %q", mockStream.recvd[0].ContextName)
	}
	if mockStream.recvd[0].KubeconfigPath != "/path/to/initial/kubeconfig" {
		t.Fatalf("expected kubeconfig path, got %q", mockStream.recvd[0].KubeconfigPath)
	}
}

// TestHostPluginServer_ConcurrentSubscribersReceivePublishes tests that concurrent subscribers all receive published events.
func TestHostPluginServer_ConcurrentSubscribersReceivePublishes(t *testing.T) {
	server := NewHostPluginServer(func(payload map[string]interface{}) {})
	defer server.MarkStopped()

	numSubscribers := 10
	subscribers := make([]*mockStreamWithClose, numSubscribers)
	var wg sync.WaitGroup

	// Create multiple subscribers
	for i := 0; i < numSubscribers; i++ {
		ctx, cancel := context.WithCancel(context.Background())
		subscribers[i] = &mockStreamWithClose{
			ctx:    ctx,
			cancel: cancel,
			recvd:  make([]*pb.ClusterContextChangedEvent, 0),
		}

		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			server.ClusterContextWatch(&pb.Empty{}, subscribers[idx])
		}(i)
	}

	// Give subscribers time to register
	time.Sleep(50 * time.Millisecond)

	// Publish multiple events
	for eventNum := 0; eventNum < 3; eventNum++ {
		contextName := "context-" + string(rune(eventNum))
		ok := server.PublishClusterContextChange(contextName, "/kubeconfig-"+string(rune(eventNum)))
		if !ok {
			t.Fatalf("publish %d failed", eventNum)
		}
		time.Sleep(10 * time.Millisecond)
	}

	// Close all subscribers
	for i := 0; i < numSubscribers; i++ {
		subscribers[i].close()
	}

	wg.Wait()

	// Verify all subscribers received all events (plus initial replay if there was one before first publish)
	for i := 0; i < numSubscribers; i++ {
		if len(subscribers[i].recvd) != 3 {
			t.Fatalf("subscriber %d: expected 3 events, got %d", i, len(subscribers[i].recvd))
		}
		for j := 0; j < 3; j++ {
			expectedContext := "context-" + string(rune(j))
			if subscribers[i].recvd[j].ContextName != expectedContext {
				t.Fatalf("subscriber %d event %d: expected %q, got %q", i, j, expectedContext, subscribers[i].recvd[j].ContextName)
			}
		}
	}
}

// TestHostPluginServer_SubscriberCleanupOnStreamDisconnect tests that subscribers are properly cleaned up when streams disconnect.
func TestHostPluginServer_SubscriberCleanupOnStreamDisconnect(t *testing.T) {
	server := NewHostPluginServer(func(payload map[string]interface{}) {})
	defer server.MarkStopped()

	// Create and immediately cancel a subscription
	ctx, cancel := context.WithCancel(context.Background())
	mockStream := &mockStreamWithClose{
		ctx:    ctx,
		cancel: cancel,
		recvd:  make([]*pb.ClusterContextChangedEvent, 0),
	}

	done := make(chan error, 1)
	go func() {
		done <- server.ClusterContextWatch(&pb.Empty{}, mockStream)
	}()

	// Give the subscription time to register
	time.Sleep(50 * time.Millisecond)

	// Cancel the subscription context
	cancel()

	// Wait for the watch to exit
	err := <-done
	if err == nil || err.Error() != "context canceled" {
		t.Logf("expected context canceled, got %v", err)
	}

	// Verify that the subscriber is no longer receiving publishes
	server.mu.RLock()
	numSubscribers := len(server.subscribers)
	server.mu.RUnlock()

	if numSubscribers != 0 {
		t.Fatalf("expected 0 subscribers after disconnect, found %d", numSubscribers)
	}

	// Verify that future publishes work (no subscriber leaks)
	ok := server.PublishClusterContextChange("test", "/test")
	if !ok {
		t.Fatal("expected publish to succeed after subscriber cleanup")
	}
}

// TestHostPluginServer_SubscriberCleanupOnStreamError tests that subscribers are cleaned up when send errors occur.
func TestHostPluginServer_SubscriberCleanupOnStreamError(t *testing.T) {
	server := NewHostPluginServer(func(payload map[string]interface{}) {})
	defer server.MarkStopped()

	// Create a stream that simulates an error on send
	ctx, cancel := context.WithCancel(context.Background())
	mockStream := &mockStreamWithError{
		ctx:    ctx,
		cancel: cancel,
		recvd:  make([]*pb.ClusterContextChangedEvent, 0),
	}

	done := make(chan error, 1)
	go func() {
		done <- server.ClusterContextWatch(&pb.Empty{}, mockStream)
	}()

	// Give the subscription time to register
	time.Sleep(50 * time.Millisecond)

	// Trigger an error by causing a send failure
	mockStream.failNextSend = true

	// Publish an event that will trigger the error
	server.PublishClusterContextChange("test-context", "/test/kubeconfig")

	// Wait for the watch to exit due to send error
	<-done

	// Verify that the subscriber is cleaned up
	server.mu.RLock()
	numSubscribers := len(server.subscribers)
	server.mu.RUnlock()

	if numSubscribers != 0 {
		t.Fatalf("expected 0 subscribers after send error, found %d", numSubscribers)
	}
}

// TestHostPluginServer_PublishRejectedAfterStop tests that publishes are rejected after stop.
func TestHostPluginServer_PublishRejectedAfterStop(t *testing.T) {
	server := NewHostPluginServer(func(payload map[string]interface{}) {})

	// Should be able to publish before stop
	ok := server.PublishClusterContextChange("context-1", "/kubeconfig-1")
	if !ok {
		t.Fatal("expected publish before stop to succeed")
	}

	// Stop the server
	server.MarkStopped()

	// Should reject publishes after stop
	ok = server.PublishClusterContextChange("context-2", "/kubeconfig-2")
	if ok {
		t.Fatal("expected publish after stop to fail")
	}
}

// TestHostPluginServer_LastEventReplacement tests that the last event is properly stored and replaced.
func TestHostPluginServer_LastEventReplacement(t *testing.T) {
	server := NewHostPluginServer(func(payload map[string]interface{}) {})
	defer server.MarkStopped()

	// Publish first event
	ok := server.PublishClusterContextChange("context-1", "/kubeconfig-1")
	if !ok {
		t.Fatal("first publish failed")
	}

	// Publish second event (should replace the first)
	ok = server.PublishClusterContextChange("context-2", "/kubeconfig-2")
	if !ok {
		t.Fatal("second publish failed")
	}

	// Publish third event (should replace the second)
	ok = server.PublishClusterContextChange("context-3", "/kubeconfig-3")
	if !ok {
		t.Fatal("third publish failed")
	}

	// New subscriber should only receive the latest event on initial subscription
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	mockStream := &mockStream{
		ctx:   ctx,
		recvd: make([]*pb.ClusterContextChangedEvent, 0),
	}

	server.ClusterContextWatch(&pb.Empty{}, mockStream)

	if len(mockStream.recvd) != 1 {
		t.Fatalf("expected 1 replay event, got %d", len(mockStream.recvd))
	}
	if mockStream.recvd[0].ContextName != "context-3" {
		t.Fatalf("expected context-3, got %q", mockStream.recvd[0].ContextName)
	}
}

// TestHostPluginServer_RaceSubscribePublish tests concurrent subscribe and publish operations.
func TestHostPluginServer_RaceSubscribePublish(t *testing.T) {
	server := NewHostPluginServer(func(payload map[string]interface{}) {})
	defer server.MarkStopped()

	var wg sync.WaitGroup
	numGoroutines := 20
	numPublishes := 10

	// Spawn goroutines that subscribe and unsubscribe rapidly
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			ctx, cancel := context.WithCancel(context.Background())
			mockStream := &mockStreamWithClose{
				ctx:    ctx,
				cancel: cancel,
				recvd:  make([]*pb.ClusterContextChangedEvent, 0),
			}

			done := make(chan error, 1)
			go func() {
				done <- server.ClusterContextWatch(&pb.Empty{}, mockStream)
			}()

			// Let it run for a bit
			time.Sleep(time.Duration(id) * time.Millisecond)
			cancel()
			<-done
		}(i)
	}

	// Spawn goroutines that publish rapidly
	for i := 0; i < numPublishes; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			server.PublishClusterContextChange("context-"+string(rune(id)), "/kubeconfig-"+string(rune(id)))
		}(i)
	}

	wg.Wait()

	// Verify final state: no subscribers left and server is responsive
	server.mu.RLock()
	numSubscribers := len(server.subscribers)
	server.mu.RUnlock()

	if numSubscribers != 0 {
		t.Fatalf("expected 0 subscribers after concurrent operations, found %d", numSubscribers)
	}
}

// TestHostPluginServer_SlowSubscriberGetsLatestNotStaleEvent tests that when a subscriber's
// channel buffer (size 1) is already full because it hasn't drained the previous event yet,
// a new publish replaces the stale queued event instead of being silently dropped — so the
// subscriber never gets stuck delivering a superseded context. This registers a subscriber
// channel directly (bypassing ClusterContextWatch's own draining goroutine) so the buffer-full
// path is exercised deterministically instead of racing a real stream loop.
func TestHostPluginServer_SlowSubscriberGetsLatestNotStaleEvent(t *testing.T) {
	server := NewHostPluginServer(func(payload map[string]interface{}) {})
	defer server.MarkStopped()

	server.mu.Lock()
	id := server.nextSubscriberID
	server.nextSubscriberID++
	ch := make(chan *pb.ClusterContextChangedEvent, 1)
	server.subscribers[id] = ch
	server.mu.Unlock()

	if ok := server.PublishClusterContextChange("context-stale", "/kubeconfig-stale"); !ok {
		t.Fatal("first publish failed")
	}
	if ok := server.PublishClusterContextChange("context-latest", "/kubeconfig-latest"); !ok {
		t.Fatal("second publish failed")
	}

	select {
	case event := <-ch:
		if event.ContextName != "context-latest" {
			t.Fatalf("expected subscriber's pending event to be the latest published context, got %q", event.ContextName)
		}
	default:
		t.Fatal("expected a pending event in the subscriber's channel, found none")
	}
}

// Mock stream implementations for testing

type mockStream struct {
	ctx   context.Context
	recvd []*pb.ClusterContextChangedEvent
	mu    sync.Mutex
}

func (m *mockStream) Send(event *pb.ClusterContextChangedEvent) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.recvd = append(m.recvd, event)
	return nil
}

func (m *mockStream) Context() context.Context {
	return m.ctx
}

func (m *mockStream) SendMsg(msg interface{}) error {
	return nil
}

func (m *mockStream) RecvMsg(msg interface{}) error {
	return nil
}

func (m *mockStream) SetHeader(md metadata.MD) error {
	return nil
}

func (m *mockStream) SendHeader(md metadata.MD) error {
	return nil
}

func (m *mockStream) SetTrailer(md metadata.MD) {
}

type mockStreamWithClose struct {
	ctx       context.Context
	cancel    context.CancelFunc
	recvd     []*pb.ClusterContextChangedEvent
	mu        sync.Mutex
	closeOnce sync.Once
}

func (m *mockStreamWithClose) Send(event *pb.ClusterContextChangedEvent) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.recvd = append(m.recvd, event)
	return nil
}

func (m *mockStreamWithClose) Context() context.Context {
	return m.ctx
}

func (m *mockStreamWithClose) SendMsg(msg interface{}) error {
	return nil
}

func (m *mockStreamWithClose) RecvMsg(msg interface{}) error {
	return nil
}

func (m *mockStreamWithClose) SetHeader(md metadata.MD) error {
	return nil
}

func (m *mockStreamWithClose) SendHeader(md metadata.MD) error {
	return nil
}

func (m *mockStreamWithClose) SetTrailer(md metadata.MD) {
}

func (m *mockStreamWithClose) close() {
	m.closeOnce.Do(func() {
		if m.cancel != nil {
			m.cancel()
		}
	})
}

type mockStreamWithError struct {
	ctx          context.Context
	cancel       context.CancelFunc
	recvd        []*pb.ClusterContextChangedEvent
	mu           sync.Mutex
	failNextSend bool
}

func (m *mockStreamWithError) Send(event *pb.ClusterContextChangedEvent) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.failNextSend {
		m.failNextSend = false
		return context.Canceled
	}
	m.recvd = append(m.recvd, event)
	return nil
}

func (m *mockStreamWithError) Context() context.Context {
	return m.ctx
}

func (m *mockStreamWithError) SendMsg(msg interface{}) error {
	return nil
}

func (m *mockStreamWithError) RecvMsg(msg interface{}) error {
	return nil
}

func (m *mockStreamWithError) SetHeader(md metadata.MD) error {
	return nil
}

func (m *mockStreamWithError) SendHeader(md metadata.MD) error {
	return nil
}

func (m *mockStreamWithError) SetTrailer(md metadata.MD) {
	_ = md
}

// mockNsStream is the ActiveNamespacesWatch analogue of mockStream.
type mockNsStream struct {
	ctx   context.Context
	recvd []*pb.ActiveNamespacesChangedEvent
	mu    sync.Mutex
}

func (m *mockNsStream) Send(event *pb.ActiveNamespacesChangedEvent) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.recvd = append(m.recvd, event)
	return nil
}

func (m *mockNsStream) Context() context.Context {
	return m.ctx
}

func (m *mockNsStream) SendMsg(msg interface{}) error {
	return nil
}

func (m *mockNsStream) RecvMsg(msg interface{}) error {
	return nil
}

func (m *mockNsStream) SetHeader(md metadata.MD) error {
	return nil
}

func (m *mockNsStream) SendHeader(md metadata.MD) error {
	return nil
}

func (m *mockNsStream) SetTrailer(md metadata.MD) {
}

// TestHostPluginServer_ActiveNamespacesSubscriptionReplay tests that new subscribers
// receive the last published active-namespaces event immediately.
func TestHostPluginServer_ActiveNamespacesSubscriptionReplay(t *testing.T) {
	server := NewHostPluginServer(func(payload map[string]interface{}) {})
	defer server.MarkStopped()

	server.PublishActiveNamespacesChange([]string{"default", "kube-system"})

	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	stream := &mockNsStream{ctx: ctx, recvd: make([]*pb.ActiveNamespacesChangedEvent, 0)}
	server.ActiveNamespacesWatch(&pb.Empty{}, stream)

	if len(stream.recvd) != 1 {
		t.Fatalf("expected 1 event from replay, got %d", len(stream.recvd))
	}
	if got := stream.recvd[0].Namespaces; len(got) != 2 || got[0] != "default" || got[1] != "kube-system" {
		t.Fatalf("expected [default kube-system], got %v", got)
	}
}

// TestHostPluginServer_ActiveNamespacesLastEventReplacement tests that only the latest
// active-namespaces event is replayed to a new subscriber.
func TestHostPluginServer_ActiveNamespacesLastEventReplacement(t *testing.T) {
	server := NewHostPluginServer(func(payload map[string]interface{}) {})
	defer server.MarkStopped()

	server.PublishActiveNamespacesChange([]string{"ns1"})
	server.PublishActiveNamespacesChange([]string{"ns2"})
	server.PublishActiveNamespacesChange([]string{"ns3", "ns4"})

	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	stream := &mockNsStream{ctx: ctx, recvd: make([]*pb.ActiveNamespacesChangedEvent, 0)}
	server.ActiveNamespacesWatch(&pb.Empty{}, stream)

	if len(stream.recvd) != 1 {
		t.Fatalf("expected 1 replay event, got %d", len(stream.recvd))
	}
	if got := stream.recvd[0].Namespaces; len(got) != 2 || got[0] != "ns3" || got[1] != "ns4" {
		t.Fatalf("expected [ns3 ns4], got %v", got)
	}
}

// TestHostPluginServer_ActiveNamespacesPublishRejectedAfterStop tests that publishes
// after MarkStopped are rejected, mirroring PublishClusterContextChange's behavior.
func TestHostPluginServer_ActiveNamespacesPublishRejectedAfterStop(t *testing.T) {
	server := NewHostPluginServer(func(payload map[string]interface{}) {})

	if ok := server.PublishActiveNamespacesChange([]string{"ns1"}); !ok {
		t.Fatal("expected publish before stop to succeed")
	}

	server.MarkStopped()

	if ok := server.PublishActiveNamespacesChange([]string{"ns2"}); ok {
		t.Fatal("expected publish after stop to fail")
	}
}

// TestHostPluginServer_ActiveNamespacesConcurrentSubscribersReceivePublishes tests that
// concurrent subscribers all receive published active-namespaces events.
func TestHostPluginServer_ActiveNamespacesConcurrentSubscribersReceivePublishes(t *testing.T) {
	server := NewHostPluginServer(func(payload map[string]interface{}) {})
	defer server.MarkStopped()

	numSubscribers := 10
	ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer cancel()

	streams := make([]*mockNsStream, numSubscribers)
	var wg sync.WaitGroup
	for i := 0; i < numSubscribers; i++ {
		streams[i] = &mockNsStream{ctx: ctx, recvd: make([]*pb.ActiveNamespacesChangedEvent, 0)}
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			server.ActiveNamespacesWatch(&pb.Empty{}, streams[idx])
		}(i)
	}

	// Give subscribers a moment to register before publishing.
	time.Sleep(20 * time.Millisecond)
	server.PublishActiveNamespacesChange([]string{"default"})

	wg.Wait()

	for i, s := range streams {
		s.mu.Lock()
		n := len(s.recvd)
		s.mu.Unlock()
		if n == 0 {
			t.Errorf("subscriber %d received no events", i)
		}
	}
}

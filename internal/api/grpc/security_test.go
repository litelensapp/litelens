package grpc

import (
	"context"
	"testing"
	"time"

	"github.com/litelensapp/litelens/packages/core/pb"
	"github.com/litelensapp/litelens/packages/core/util"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

// injectPluginID returns a context with a plugin ID injected for testing purposes.
func injectPluginID(ctx context.Context, pluginID string) context.Context {
	return context.WithValue(ctx, contextKeyPluginID{}, pluginID)
}

// ============================================================
// Auth Interceptor Tests (item 1)
// ============================================================

// TestUnaryServerInterceptor_MissingAuthToken verifies that unary requests without
// an authorization header return codes.Unauthenticated.
func TestUnaryServerInterceptor_MissingAuthToken(t *testing.T) {
	manager := NewAuthTokenManager()
	interceptor := UnaryServerInterceptor(manager)

	handler := func(ctx context.Context, req any) (any, error) {
		return nil, nil
	}

	// Call without authorization header
	ctx := context.Background()
	_, err := interceptor(ctx, nil, &grpc.UnaryServerInfo{}, handler)

	if err == nil {
		t.Fatal("expected error for missing auth token")
	}

	st, ok := status.FromError(err)
	if !ok || st.Code() != codes.Unauthenticated {
		t.Fatalf("expected codes.Unauthenticated, got %v", err)
	}
}

// TestUnaryServerInterceptor_MalformedAuthHeader verifies that malformed auth headers
// (missing "bearer " prefix) return codes.Unauthenticated.
func TestUnaryServerInterceptor_MalformedAuthHeader(t *testing.T) {
	manager := NewAuthTokenManager()
	interceptor := UnaryServerInterceptor(manager)

	handler := func(ctx context.Context, req any) (any, error) {
		return nil, nil
	}

	// Call with malformed auth header (missing "bearer " prefix)
	md := metadata.Pairs("authorization", "invalid-token-no-bearer-prefix")
	ctx := metadata.NewIncomingContext(context.Background(), md)

	_, err := interceptor(ctx, nil, &grpc.UnaryServerInfo{}, handler)

	if err == nil {
		t.Fatal("expected error for malformed auth header")
	}

	st, ok := status.FromError(err)
	if !ok || st.Code() != codes.Unauthenticated {
		t.Fatalf("expected codes.Unauthenticated, got %v", err)
	}
}

// TestUnaryServerInterceptor_UnknownToken verifies that unknown tokens return
// codes.Unauthenticated (same error as missing, no fingerprinting).
func TestUnaryServerInterceptor_UnknownToken(t *testing.T) {
	manager := NewAuthTokenManager()
	interceptor := UnaryServerInterceptor(manager)

	handler := func(ctx context.Context, req any) (any, error) {
		return nil, nil
	}

	// Call with valid bearer token format, but unknown token
	md := metadata.Pairs("authorization", "bearer unknown-token-not-registered")
	ctx := metadata.NewIncomingContext(context.Background(), md)

	_, err := interceptor(ctx, nil, &grpc.UnaryServerInfo{}, handler)

	if err == nil {
		t.Fatal("expected error for unknown token")
	}

	st, ok := status.FromError(err)
	if !ok || st.Code() != codes.Unauthenticated {
		t.Fatalf("expected codes.Unauthenticated, got %v", err)
	}
}

// TestUnaryServerInterceptor_ValidToken verifies that a registered token allows
// the request to proceed and injects the plugin ID into the context.
func TestUnaryServerInterceptor_ValidToken(t *testing.T) {
	manager := NewAuthTokenManager()
	const testToken = "test-token-12345"
	const testPluginID = "test-plugin"
	manager.RegisterToken(testToken, testPluginID)

	interceptor := UnaryServerInterceptor(manager)

	var capturedCtx context.Context
	handler := func(ctx context.Context, req any) (any, error) {
		capturedCtx = ctx
		return "success", nil
	}

	md := metadata.Pairs("authorization", "bearer "+testToken)
	ctx := metadata.NewIncomingContext(context.Background(), md)

	result, err := interceptor(ctx, nil, &grpc.UnaryServerInfo{}, handler)

	if err != nil {
		t.Fatalf("expected no error for valid token, got %v", err)
	}

	if result != "success" {
		t.Fatalf("expected success response, got %v", result)
	}

	// Verify plugin ID was injected
	pluginID := PluginIDFromContext(capturedCtx)
	if pluginID != testPluginID {
		t.Fatalf("expected plugin ID %q in context, got %q", testPluginID, pluginID)
	}
}

// TestStreamServerInterceptor_MissingAuthToken verifies that stream requests without
// an authorization header return codes.Unauthenticated.
func TestStreamServerInterceptor_MissingAuthToken(t *testing.T) {
	manager := NewAuthTokenManager()
	interceptor := StreamServerInterceptor(manager)

	handler := func(srv any, ss grpc.ServerStream) error {
		return nil
	}

	mockStream := &mockServerStream{
		ctx: context.Background(),
	}

	err := interceptor(nil, mockStream, &grpc.StreamServerInfo{}, handler)

	if err == nil {
		t.Fatal("expected error for missing auth token")
	}

	st, ok := status.FromError(err)
	if !ok || st.Code() != codes.Unauthenticated {
		t.Fatalf("expected codes.Unauthenticated, got %v", err)
	}
}

// TestStreamServerInterceptor_ValidToken verifies that a registered token allows
// the stream to proceed and injects the plugin ID into the context.
func TestStreamServerInterceptor_ValidToken(t *testing.T) {
	manager := NewAuthTokenManager()
	const testToken = "test-token-stream"
	const testPluginID = "test-plugin-stream"
	manager.RegisterToken(testToken, testPluginID)

	interceptor := StreamServerInterceptor(manager)

	var capturedCtx context.Context
	handler := func(srv any, ss grpc.ServerStream) error {
		capturedCtx = ss.Context()
		return nil
	}

	md := metadata.Pairs("authorization", "bearer "+testToken)
	ctx := metadata.NewIncomingContext(context.Background(), md)
	mockStream := &mockServerStream{
		ctx: ctx,
	}

	err := interceptor(nil, mockStream, &grpc.StreamServerInfo{}, handler)

	if err != nil {
		t.Fatalf("expected no error for valid token, got %v", err)
	}

	// Verify plugin ID was injected
	pluginID := PluginIDFromContext(capturedCtx)
	if pluginID != testPluginID {
		t.Fatalf("expected plugin ID %q in context, got %q", testPluginID, pluginID)
	}
}

// ============================================================
// Topic Validation Tests (item 2)
// ============================================================

// TestTopicValidation_EmptyString verifies that empty topic names are rejected.
func TestTopicValidation_EmptyString(t *testing.T) {
	if isValidTopic("") {
		t.Fatal("expected empty topic to be invalid")
	}
}

// TestTopicValidation_ValidTopics verifies that properly formatted topics pass validation.
func TestTopicValidation_ValidTopics(t *testing.T) {
	validTopics := []string{
		"cluster.context",
		"namespaces.active",
		"plugins.helm.install",
		"foo",
		"foo.bar.baz",
		"foo-bar_baz",
		"foo123",
		"123foo",
	}

	for _, topic := range validTopics {
		if !isValidTopic(topic) {
			t.Fatalf("expected %q to be valid", topic)
		}
	}
}

// TestTopicValidation_InvalidPatterns verifies that topics with invalid characters are rejected.
func TestTopicValidation_InvalidPatterns(t *testing.T) {
	invalidTopics := []string{
		"foo bar",  // spaces
		"foo/bar",  // slash
		"foo@bar",  // at sign
		"foo#bar",  // hash
		"/foo",     // leading slash
		"foo/",     // trailing slash
		"foo\\bar", // backslash
	}

	for _, topic := range invalidTopics {
		if isValidTopic(topic) {
			t.Fatalf("expected %q to be invalid", topic)
		}
	}

	// Note: "foo..bar" is currently allowed by the regex (^[a-zA-Z0-9._-]+$) but
	// could be considered a semantic issue. This is acceptable for now.
}

// TestSubscribe_InvalidTopic verifies that Subscribe rejects invalid topic names.
func TestSubscribe_InvalidTopic(t *testing.T) {
	authManager := NewAuthTokenManager()
	const testToken = "test-token"
	const testPluginID = "test-plugin"
	authManager.RegisterToken(testToken, testPluginID)

	server := NewHostPluginServer(func(payload map[string]any) {}, authManager)
	defer server.MarkStopped()

	invalidTopics := []string{
		"",
		"foo bar",
		"foo/bar",
	}

	for _, topic := range invalidTopics {
		ctx := injectPluginID(context.Background(), testPluginID)
		err := server.Subscribe(&pb.SubscribeRequest{Topic: topic}, &mockSubscribeStream{ctx: ctx})

		if err == nil {
			t.Fatalf("expected error for invalid topic %q", topic)
		}

		st, ok := status.FromError(err)
		if !ok || st.Code() != codes.InvalidArgument {
			t.Fatalf("expected codes.InvalidArgument for topic %q, got %v", topic, err)
		}
	}
}

// TestPublish_InvalidTopic verifies that Publish rejects invalid topic names.
func TestPublish_InvalidTopic(t *testing.T) {
	authManager := NewAuthTokenManager()
	const testToken = "test-token"
	const testPluginID = "test-plugin"
	authManager.RegisterToken(testToken, testPluginID)

	server := NewHostPluginServer(func(payload map[string]any) {}, authManager)
	defer server.MarkStopped()

	invalidTopics := []string{
		"",
		"foo bar",
		"foo/bar",
	}

	for _, topic := range invalidTopics {
		ctx := injectPluginID(context.Background(), testPluginID)
		_, err := server.Publish(ctx, &pb.PublishRequest{Topic: topic, PayloadJson: "{}"})

		if err == nil {
			t.Fatalf("expected error for invalid topic %q", topic)
		}

		st, ok := status.FromError(err)
		if !ok || st.Code() != codes.InvalidArgument {
			t.Fatalf("expected codes.InvalidArgument for topic %q, got %v", topic, err)
		}
	}
}

// ============================================================
// Reserved Namespace Enforcement Tests (item 3)
// ============================================================

// TestPublish_ReservedNamespaceRejected verifies that a plugin cannot publish
// to reserved host-owned topics (cluster.*, namespaces.*) — those are host-only.
func TestPublish_ReservedNamespaceRejected(t *testing.T) {
	authManager := NewAuthTokenManager()
	const testToken = "test-token"
	const testPluginID = "helm"
	authManager.RegisterToken(testToken, testPluginID)

	server := NewHostPluginServer(func(payload map[string]any) {}, authManager)
	defer server.MarkStopped()

	ctx := injectPluginID(context.Background(), testPluginID)

	for _, topic := range []string{string(util.EventTopicClusterContext), string(util.EventTopicNamespacesActive)} {
		_, err := server.Publish(ctx, &pb.PublishRequest{
			Topic:       topic,
			PayloadJson: "{}",
		})
		if err == nil {
			t.Fatalf("expected error publishing to reserved topic %q, got nil", topic)
		}
		if status.Code(err) != codes.PermissionDenied {
			t.Errorf("topic %q: expected codes.PermissionDenied, got %v", topic, status.Code(err))
		}
	}
}

// TestPublish_PluginOwnNamespaceAllowed verifies that a plugin can publish
// to topics under plugins.<its-own-id>.*.
func TestPublish_PluginOwnNamespaceAllowed(t *testing.T) {
	authManager := NewAuthTokenManager()
	const testToken = "test-token"
	const testPluginID = "helm"
	authManager.RegisterToken(testToken, testPluginID)

	server := NewHostPluginServer(func(payload map[string]any) {}, authManager)
	defer server.MarkStopped()

	ctx := injectPluginID(context.Background(), testPluginID)
	_, err := server.Publish(ctx, &pb.PublishRequest{
		Topic:       "plugins.helm.chart-installed",
		PayloadJson: "{}",
	})

	if err != nil {
		t.Fatalf("expected plugin to publish to its own namespace, got %v", err)
	}
}

// TestPublish_OtherPluginNamespaceRejected verifies that a plugin cannot
// publish to another plugin's namespace.
func TestPublish_OtherPluginNamespaceRejected(t *testing.T) {
	authManager := NewAuthTokenManager()
	const testToken = "test-token"
	const testPluginID = "helm"
	authManager.RegisterToken(testToken, testPluginID)

	server := NewHostPluginServer(func(payload map[string]any) {}, authManager)
	defer server.MarkStopped()

	ctx := injectPluginID(context.Background(), testPluginID)
	_, err := server.Publish(ctx, &pb.PublishRequest{
		Topic:       "plugins.other-plugin.something",
		PayloadJson: "{}",
	})

	if err == nil {
		t.Fatal("expected plugin to be denied publishing to other plugin's namespace")
	}

	st, ok := status.FromError(err)
	if !ok || st.Code() != codes.PermissionDenied {
		t.Fatalf("expected codes.PermissionDenied, got %v", err)
	}
}

// ============================================================
// PubSubBroker Late-Joiner Replay Tests (item 5)
// ============================================================

// TestPubSubBroker_HostTopicLateJoinerGetsReplay verifies that new subscribers
// to host-owned topics receive the last published message immediately.
func TestPubSubBroker_HostTopicLateJoinerGetsReplay(t *testing.T) {
	broker := NewPubSubBroker()

	// Publish to a host-owned topic
	msg := &pb.PubSubMessage{
		Topic:       string(util.EventTopicClusterContext),
		Source:      "host",
		Timestamp:   time.Now().Format(time.RFC3339),
		PayloadJson: `{"contextName":"test"}`,
	}
	broker.publish(string(util.EventTopicClusterContext), msg)
	broker.setLastMessage(string(util.EventTopicClusterContext), msg)

	// Late joiner subscribes
	subID, _ := broker.registerSubscriber(string(util.EventTopicClusterContext))
	defer broker.unregisterSubscriber(string(util.EventTopicClusterContext), subID)

	// Should receive last message immediately
	lastMsg := broker.getLastMessage(string(util.EventTopicClusterContext))
	if lastMsg == nil {
		t.Fatal("expected last message to be cached for host-owned topic")
	}
	if lastMsg.PayloadJson != `{"contextName":"test"}` {
		t.Fatalf("expected cached message payload, got %s", lastMsg.PayloadJson)
	}
}

// TestPubSubBroker_PluginTopicNoReplay verifies that new subscribers to plugin-owned
// topics do NOT get replay (no cached last message).
func TestPubSubBroker_PluginTopicNoReplay(t *testing.T) {
	broker := NewPubSubBroker()

	// Publish to a plugin-owned topic
	msg := &pb.PubSubMessage{
		Topic:       "plugins.helm.installed",
		Source:      "helm",
		Timestamp:   time.Now().Format(time.RFC3339),
		PayloadJson: `{"chart":"nginx"}`,
	}
	broker.publish("plugins.helm.installed", msg)
	// Note: NOT calling setLastMessage for plugin-owned topics

	// Late joiner subscribes
	subID, _ := broker.registerSubscriber("plugins.helm.installed")
	defer broker.unregisterSubscriber("plugins.helm.installed", subID)

	// Should NOT receive replay
	lastMsg := broker.getLastMessage("plugins.helm.installed")
	if lastMsg != nil {
		t.Fatal("expected no cached message for plugin-owned topic")
	}
}

// ============================================================
// PubSubBroker Buffer-1 Drop-Oldest Tests (item 7)
// ============================================================

// TestPubSubBroker_BufferFull_DropsOldest verifies that when a subscriber's
// buffer is full, a new publish replaces the oldest pending message.
func TestPubSubBroker_BufferFull_DropsOldest(t *testing.T) {
	broker := NewPubSubBroker()

	subID, ch := broker.registerSubscriber("test.topic")
	defer broker.unregisterSubscriber("test.topic", subID)

	// Fill the buffer
	msg1 := &pb.PubSubMessage{Topic: "test.topic", PayloadJson: `{"seq":1}`}
	select {
	case ch <- msg1:
	default:
		t.Fatal("first message should fit in buffer")
	}

	// Buffer is now full. Publish should drop msg1 and queue msg2
	msg2 := &pb.PubSubMessage{Topic: "test.topic", PayloadJson: `{"seq":2}`}
	broker.publish("test.topic", msg2)

	// Read what's in the buffer
	select {
	case received := <-ch:
		if received.PayloadJson != `{"seq":2}` {
			t.Fatalf("expected msg2, got %s", received.PayloadJson)
		}
	default:
		t.Fatal("expected a message in the buffer")
	}

	// Buffer should be empty now
	select {
	case <-ch:
		t.Fatal("expected only one message in buffer")
	default:
		// OK
	}
}

// ============================================================
// Mock Helpers
// ============================================================

type mockServerStream struct {
	ctx context.Context
}

func (m *mockServerStream) SetHeader(metadata.MD) error  { return nil }
func (m *mockServerStream) SendHeader(metadata.MD) error { return nil }
func (m *mockServerStream) SetTrailer(metadata.MD)       {}
func (m *mockServerStream) Context() context.Context     { return m.ctx }
func (m *mockServerStream) SendMsg(msg any) error        { return nil }
func (m *mockServerStream) RecvMsg(msg any) error        { return nil }

type mockSubscribeStream struct {
	ctx   context.Context
	recvd []*pb.PubSubMessage
}

func (m *mockSubscribeStream) Send(msg *pb.PubSubMessage) error {
	m.recvd = append(m.recvd, msg)
	return nil
}

func (m *mockSubscribeStream) SetHeader(metadata.MD) error  { return nil }
func (m *mockSubscribeStream) SendHeader(metadata.MD) error { return nil }
func (m *mockSubscribeStream) SetTrailer(metadata.MD)       {}
func (m *mockSubscribeStream) Context() context.Context     { return m.ctx }
func (m *mockSubscribeStream) SendMsg(msg any) error        { return nil }
func (m *mockSubscribeStream) RecvMsg(msg any) error        { return nil }

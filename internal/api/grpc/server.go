package grpc

import (
	"context"
	"fmt"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/litelensapp/litelens/packages/core/pb"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// HostPluginServer implements the gRPC Plugin service with native pub/sub support
// via Subscribe/Publish.
type HostPluginServer struct {
	pb.UnimplementedPluginServer
	mu sync.RWMutex

	// Native pub/sub broker
	broker *PubSubBroker

	stopped     bool // flag to prevent publishing during shutdown
	eventEmitFn func(payload map[string]any)
	authManager *AuthTokenManager
}

// PubSubBroker manages topics and subscriber channels for pub/sub messaging.
// It maintains a lastMessage cache for host-owned topics (cluster.*, namespaces.*).
type PubSubBroker struct {
	mu          sync.RWMutex
	topics      map[string]map[int]chan *pb.PubSubMessage // topic -> subscriberID -> channel
	nextID      map[string]int                            // topic -> next subscriber ID
	lastMessage map[string]*pb.PubSubMessage              // topic -> last message (host topics only)
}

// NewPubSubBroker creates a new pub/sub broker.
func NewPubSubBroker() *PubSubBroker {
	return &PubSubBroker{
		topics:      make(map[string]map[int]chan *pb.PubSubMessage),
		nextID:      make(map[string]int),
		lastMessage: make(map[string]*pb.PubSubMessage),
	}
}

// NewHostPluginServer creates a new HostPluginServer.
func NewHostPluginServer(eventEmitFn func(payload map[string]any), authManager *AuthTokenManager) *HostPluginServer {
	return &HostPluginServer{
		broker:      NewPubSubBroker(),
		eventEmitFn: eventEmitFn,
		authManager: authManager,
	}
}

// topicRegex validates topic names: alphanumeric, dots, underscores, hyphens.
var topicRegex = regexp.MustCompile(`^[a-zA-Z0-9._-]+$`)

// isValidTopic checks if a topic name is valid.
func isValidTopic(topic string) bool {
	return topic != "" && topicRegex.MatchString(topic)
}

// isHostOwnedTopic checks if a topic is host-owned (e.g., cluster.*, namespaces.*).
func isHostOwnedTopic(topic string) bool {
	return len(topic) > 0 && (topic[0:min(8, len(topic))] == "cluster." || topic[0:min(11, len(topic))] == "namespaces.")
}

// Subscribe handles native pub/sub subscriptions for a given topic.
func (s *HostPluginServer) Subscribe(req *pb.SubscribeRequest, stream pb.Plugin_SubscribeServer) error {
	topic := req.Topic
	ctx := stream.Context()

	// Validate topic
	if !isValidTopic(topic) {
		return status.Error(codes.InvalidArgument, "invalid topic format")
	}

	// Register subscriber
	subscriberID, ch := s.broker.registerSubscriber(topic)

	// Send last message immediately if it exists (host-owned topics only)
	lastMsg := s.broker.getLastMessage(topic)
	if lastMsg != nil {
		if err := stream.Send(lastMsg); err != nil {
			s.broker.unregisterSubscriber(topic, subscriberID)
			return err
		}
	}

	// Stream loop
	for {
		select {
		case <-ctx.Done():
			s.broker.unregisterSubscriber(topic, subscriberID)
			return ctx.Err()
		case msg := <-ch:
			if err := stream.Send(msg); err != nil {
				s.broker.unregisterSubscriber(topic, subscriberID)
				return err
			}
		}
	}
}

// Publish publishes a message to a topic.
// Plugins may only publish to plugins.<their-own-pluginID>.* topics.
// The host may publish to any topic.
func (s *HostPluginServer) Publish(ctx context.Context, req *pb.PublishRequest) (*pb.Empty, error) {
	topic := req.Topic
	pluginID := PluginIDFromContext(ctx)

	// Validate topic
	if !isValidTopic(topic) {
		return nil, status.Error(codes.InvalidArgument, "invalid topic format")
	}

	s.mu.RLock()
	stopped := s.stopped
	s.mu.RUnlock()
	if stopped {
		return nil, status.Error(codes.Unavailable, "server is shutting down")
	}

	// Authorization check: plugins can only publish under plugins.<their-pluginID>.*
	// Reserved namespaces (cluster.*, namespaces.*) are host-only.
	if pluginID != "" {
		if isHostOwnedTopic(topic) {
			return nil, status.Error(codes.PermissionDenied, "plugins cannot publish to reserved namespaces")
		}
		allowedPrefix := fmt.Sprintf("plugins.%s.", pluginID)
		if !strings.HasPrefix(topic, allowedPrefix) {
			return nil, status.Error(codes.PermissionDenied, "plugins can only publish to their own namespace")
		}
	}

	// Create pub/sub message
	msg := &pb.PubSubMessage{
		Topic:       topic,
		Source:      pluginID,
		Timestamp:   time.Now().Format(time.RFC3339),
		PayloadJson: req.PayloadJson,
	}

	// Publish to all subscribers (non-blocking, buffer-1, drop-oldest)
	s.broker.publish(topic, msg)

	// Update last-message cache only for host-owned topics
	if isHostOwnedTopic(topic) {
		s.broker.setLastMessage(topic, msg)
	}

	return &pb.Empty{}, nil
}

// PublishToHost is called by host-internal code to publish cluster/namespace changes.
// It publishes to the broker without going through the plugin-facing gRPC RPC.
// Returns false if the topic is invalid or the server is shutting down.
func (s *HostPluginServer) PublishToHost(topic string, payloadJSON string) bool {
	if !isValidTopic(topic) {
		return false
	}

	s.mu.RLock()
	stopped := s.stopped
	s.mu.RUnlock()
	if stopped {
		return false
	}

	msg := &pb.PubSubMessage{
		Topic:       topic,
		Source:      "host",
		Timestamp:   time.Now().Format(time.RFC3339),
		PayloadJson: payloadJSON,
	}

	s.broker.publish(topic, msg)

	// Cache for host-owned topics
	if isHostOwnedTopic(topic) {
		s.broker.setLastMessage(topic, msg)
	}
	return true
}

// Broker helper methods

// registerSubscriber registers a subscriber for a topic and returns the subscriber ID and channel.
func (b *PubSubBroker) registerSubscriber(topic string) (int, chan *pb.PubSubMessage) {
	b.mu.Lock()
	defer b.mu.Unlock()

	if b.topics[topic] == nil {
		b.topics[topic] = make(map[int]chan *pb.PubSubMessage)
		b.nextID[topic] = 0
	}

	subscriberID := b.nextID[topic]
	b.nextID[topic]++

	ch := make(chan *pb.PubSubMessage, 1)
	b.topics[topic][subscriberID] = ch

	return subscriberID, ch
}

// unregisterSubscriber removes a subscriber from a topic.
func (b *PubSubBroker) unregisterSubscriber(topic string, subscriberID int) {
	b.mu.Lock()
	defer b.mu.Unlock()

	if subs, ok := b.topics[topic]; ok {
		if ch, ok := subs[subscriberID]; ok {
			close(ch)
			delete(subs, subscriberID)
		}
		if len(subs) == 0 {
			delete(b.topics, topic)
		}
	}
}

// publish publishes a message to all subscribers of a topic (non-blocking, drop-oldest on full).
func (b *PubSubBroker) publish(topic string, msg *pb.PubSubMessage) {
	// Hold the read lock for the entire send loop (sends are non-blocking via
	// select/default, so this is cheap) — this prevents unregisterSubscriber,
	// which takes the write lock, from closing a channel or mutating the
	// topic's subscriber map concurrently with this range/send.
	b.mu.RLock()
	defer b.mu.RUnlock()

	subs, ok := b.topics[topic]
	if !ok || len(subs) == 0 {
		return
	}

	for _, ch := range subs {
		select {
		case ch <- msg:
		default:
			// Channel full, drop oldest message and try again
			select {
			case <-ch:
			default:
			}
			select {
			case ch <- msg:
			default:
			}
		}
	}
}

// getLastMessage returns the last message for a topic (if cached).
func (b *PubSubBroker) getLastMessage(topic string) *pb.PubSubMessage {
	b.mu.RLock()
	defer b.mu.RUnlock()
	return b.lastMessage[topic]
}

// setLastMessage caches the last message for a topic.
func (b *PubSubBroker) setLastMessage(topic string, msg *pb.PubSubMessage) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.lastMessage[topic] = msg
}

// MarkStopped marks the server as stopped, preventing new publishes.
func (s *HostPluginServer) MarkStopped() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.stopped = true
}

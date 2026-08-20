package server

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/litelensapp/litelens/packages/core/pb"
)

// HostPluginServer implements the gRPC Plugin service streaming cluster context changes.
type HostPluginServer struct {
	pb.UnimplementedPluginServer
	mu                sync.RWMutex
	subscribers       map[int]chan *pb.ClusterContextChangedEvent
	nextSubscriberID  int
	lastContextChange *pb.ClusterContextChangedEvent
	stopped           bool // flag to prevent publishing during shutdown
	eventEmitFn       func(payload map[string]interface{})
}

// NewHostPluginServer creates a new HostPluginServer.
func NewHostPluginServer(eventEmitFn func(payload map[string]interface{})) *HostPluginServer {
	return &HostPluginServer{
		subscribers: make(map[int]chan *pb.ClusterContextChangedEvent),
		eventEmitFn: eventEmitFn,
	}
}

// ClusterContextWatch handles the streaming RPC that plugins call to subscribe to context changes.
func (s *HostPluginServer) ClusterContextWatch(req *pb.Empty, stream pb.Plugin_ClusterContextWatchServer) error {
	// Register a new subscriber
	s.mu.Lock()
	subscriberID := s.nextSubscriberID
	s.nextSubscriberID++
	ch := make(chan *pb.ClusterContextChangedEvent, 1)
	s.subscribers[subscriberID] = ch

	// Send the last context change immediately if one exists
	lastChange := s.lastContextChange
	s.mu.Unlock()

	if lastChange != nil {
		if err := stream.Send(lastChange); err != nil {
			s.mu.Lock()
			delete(s.subscribers, subscriberID)
			s.mu.Unlock()
			return err
		}
	}

	// Loop to send future changes to this subscriber
	for {
		select {
		case <-stream.Context().Done():
			s.mu.Lock()
			delete(s.subscribers, subscriberID)
			s.mu.Unlock()
			return stream.Context().Err()
		case event := <-ch:
			if err := stream.Send(event); err != nil {
				s.mu.Lock()
				delete(s.subscribers, subscriberID)
				s.mu.Unlock()
				return err
			}
		}
	}
}

// PublishClusterContextChange sends a cluster context change event to all currently subscribed plugins.
// Returns false if the server is shutting down (event not published).
//
// Event-Ordering Guarantee:
// Each subscriber is guaranteed to receive events in order. However, a subscriber that
// arrives after a publish begins may receive a stale context replay (the last published
// event) before receiving subsequent new events. This is acceptable because:
// - Single-active-context semantics mean only one context is ever valid at a time
// - A stale replay self-heals on the next actual cluster switch
// - This avoids the complexity of buffering or ordering guarantees across late-joiners
// Plugins should expect to receive occasional redundant replays and handle them idempotently
// (which they do — SetActiveContext is a no-op if the context is already set).
func (s *HostPluginServer) PublishClusterContextChange(contextName, kubeconfigPath string) bool {
	event := &pb.ClusterContextChangedEvent{
		ContextName:    contextName,
		KubeconfigPath: kubeconfigPath,
		Timestamp:      time.Now().Format(time.RFC3339),
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	// Reject publishes during shutdown
	if s.stopped {
		return false
	}

	// Store the last published event for new subscribers
	s.lastContextChange = event

	// Broadcast to all current subscribers. The channel is buffer-1 and holds only the
	// most recent pending event: if it's already full (the subscriber hasn't drained
	// the previous event yet), drop the stale one and enqueue this one in its place.
	// Dropping the newest event instead would let a subscriber get stuck delivering a
	// superseded context indefinitely, since nothing here retries a skipped publish.
	for _, ch := range s.subscribers {
		select {
		case ch <- event:
		default:
			select {
			case <-ch:
			default:
			}
			select {
			case ch <- event:
			default:
			}
		}
	}
	return true
}

// MarkStopped marks the server as stopped, preventing new publishes.
func (s *HostPluginServer) MarkStopped() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.stopped = true
}

// EmitEvent handles event emission from plugins to the host.
func (s *HostPluginServer) EmitEvent(ctx context.Context, req *pb.PluginEventRequest) (*pb.Empty, error) {
	if req.PluginId == "" || req.EventName == "" {
		return nil, fmt.Errorf("invalid event request: pluginId and eventName required")
	}
	if s.eventEmitFn == nil {
		return &pb.Empty{}, nil
	}
	var payload interface{}
	if req.PayloadJson != "" {
		if err := json.Unmarshal([]byte(req.PayloadJson), &payload); err != nil {
			return nil, fmt.Errorf("invalid payloadJson: %w", err)
		}
	}
	s.eventEmitFn(map[string]interface{}{
		"pluginId":  req.PluginId,
		"eventName": req.EventName,
		"payload":   payload,
	})
	return &pb.Empty{}, nil
}

package util

// EventTopic identifies a host<->plugin pub/sub topic over the gRPC
// PublishToHost/Subscribe channel.
type EventTopic string

const (
	// EventTopicClusterContext carries the active cluster context and its
	// kubeconfig path, pushed by the host on every cluster switch.
	EventTopicClusterContext EventTopic = "cluster.context"
	// EventTopicNamespacesActive carries the active namespace filter, pushed
	// by the host on every namespace-filter change.
	EventTopicNamespacesActive EventTopic = "namespaces.active"
)

package async

import "github.com/litelensapp/litelens/packages/core/pb"

// SubscribeStream is a stream of pub/sub messages.
type SubscribeStream interface {
	Recv() (*pb.PubSubMessage, error)
}

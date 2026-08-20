package grpc

import (
	"fmt"
	"net"

	"github.com/litelensapp/litelens/packages/core/pb"
	"google.golang.org/grpc"
)

// GRPCServerConfig manages the host's gRPC server for plugin cluster context streaming.
type GRPCServerConfig struct {
	server       *grpc.Server
	port         int
	pluginServer *HostPluginServer
	ln           net.Listener // Keep listener alive for the lifetime of the server
}

// NewGRPCServerConfig starts a new gRPC server on 127.0.0.1:0 (auto-assigned port).
func NewGRPCServerConfig(eventEmitFn func(payload map[string]interface{})) (*GRPCServerConfig, error) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return nil, fmt.Errorf("listen for gRPC: %w", err)
	}

	server := grpc.NewServer()
	pluginServer := NewHostPluginServer(eventEmitFn)
	pb.RegisterPluginServer(server, pluginServer)

	cfg := &GRPCServerConfig{
		server:       server,
		pluginServer: pluginServer,
		ln:           ln,
	}

	// Extract port from listener
	addr := ln.Addr().String()
	_, portStr, err := net.SplitHostPort(addr)
	if err != nil {
		server.Stop()
		return nil, fmt.Errorf("parse listener address: %w", err)
	}
	_, err = fmt.Sscanf(portStr, "%d", &cfg.port)
	if err != nil {
		server.Stop()
		return nil, fmt.Errorf("parse port: %w", err)
	}

	// Start serving in background
	go func() {
		if err := server.Serve(ln); err != nil && err != net.ErrClosed {
			fmt.Printf("grpc server error: %v\n", err)
		}
	}()

	return cfg, nil
}

// Port returns the assigned gRPC port.
func (c *GRPCServerConfig) Port() int {
	return c.port
}

// PluginServer returns the underlying HostPluginServer for publishing context changes.
func (c *GRPCServerConfig) PluginServer() *HostPluginServer {
	return c.pluginServer
}

// Stop gracefully stops the server. It marks the server as stopped first to prevent
// new publishes during shutdown, then waits for active streams to close.
func (c *GRPCServerConfig) Stop() {
	if c.pluginServer != nil {
		c.pluginServer.MarkStopped()
	}
	if c.server != nil {
		c.server.GracefulStop()
	}
}

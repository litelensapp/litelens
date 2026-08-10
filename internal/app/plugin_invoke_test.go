package app

import (
	"context"
	"testing"

	"google.golang.org/grpc"

	"github.com/gknguyen/litelens/internal/config"
	"github.com/gknguyen/litelens/internal/dto"
	"github.com/gknguyen/litelens/internal/plugin"
	"github.com/gknguyen/litelens/internal/plugin/pb"
)

// mockPluginClient implements pb.PluginClient for testing
type mockPluginClient struct {
	GetCapabilitiesFn   func(context.Context, *pb.Empty, ...grpc.CallOption) (*pb.CapabilitiesResponse, error)
	SetClusterContextFn func(context.Context, *pb.SetClusterContextRequest, ...grpc.CallOption) (*pb.Empty, error)
	InvokeFn            func(context.Context, *pb.InvokeRequest, ...grpc.CallOption) (*pb.InvokeResponse, error)
}

func (m *mockPluginClient) GetCapabilities(ctx context.Context, req *pb.Empty, opts ...grpc.CallOption) (*pb.CapabilitiesResponse, error) {
	if m.GetCapabilitiesFn != nil {
		return m.GetCapabilitiesFn(ctx, req, opts...)
	}
	return &pb.CapabilitiesResponse{Version: "mock", Ready: true}, nil
}

func (m *mockPluginClient) SetClusterContext(ctx context.Context, req *pb.SetClusterContextRequest, opts ...grpc.CallOption) (*pb.Empty, error) {
	if m.SetClusterContextFn != nil {
		return m.SetClusterContextFn(ctx, req, opts...)
	}
	return &pb.Empty{}, nil
}

func (m *mockPluginClient) Invoke(ctx context.Context, req *pb.InvokeRequest, opts ...grpc.CallOption) (*pb.InvokeResponse, error) {
	if m.InvokeFn != nil {
		return m.InvokeFn(ctx, req, opts...)
	}
	return &pb.InvokeResponse{PayloadJson: "{}", Error: ""}, nil
}

func TestInvokePluginNotInstalled(t *testing.T) {
	a := &App{
		pluginLoaders: make(map[string]*plugin.PluginLoader),
	}

	_, err := a.InvokePlugin("helm", "ListHelmCharts", "{}")
	if err == nil {
		t.Error("expected error for non-installed plugin")
	}
	if err.Error() != "plugin \"helm\" not installed" {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestInvokePluginNotReady(t *testing.T) {
	a := &App{
		pluginLoaders: make(map[string]*plugin.PluginLoader),
	}

	loader := plugin.NewPluginLoader("helm", "")
	loader.SetStatus(dto.PluginStatusInstalling)
	a.pluginLoaders["helm"] = loader

	_, err := a.InvokePlugin("helm", "ListHelmCharts", "{}")
	if err == nil {
		t.Error("expected error for non-ready plugin")
	}
}

func TestInvokePluginNoActiveContext(t *testing.T) {
	a := &App{
		pluginLoaders: make(map[string]*plugin.PluginLoader),
		activeContext: "",
		settings:      config.Settings{},
	}

	loader := plugin.NewPluginLoader("helm", "")
	loader.SetStatus(dto.PluginStatusReady)
	a.pluginLoaders["helm"] = loader

	_, err := a.InvokePlugin("helm", "ListHelmCharts", "{}")
	if err == nil {
		t.Error("expected error for no active context")
	}
	if err.Error() != "plugin \"helm\" requires a connected cluster context" {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestInvokePluginSuccess(t *testing.T) {
	a := &App{
		pluginLoaders: make(map[string]*plugin.PluginLoader),
		activeContext: "test-context",
		settings: config.Settings{
			KubeconfigPaths: []string{"/tmp/kubeconfig"},
		},
	}

	loader := plugin.NewPluginLoader("helm", "")
	loader.SetStatus(dto.PluginStatusReady)

	mockClient := &mockPluginClient{
		SetClusterContextFn: func(ctx context.Context, req *pb.SetClusterContextRequest, opts ...grpc.CallOption) (*pb.Empty, error) {
			if req.ContextName != "test-context" {
				t.Errorf("expected context name 'test-context', got %q", req.ContextName)
			}
			return &pb.Empty{}, nil
		},
		InvokeFn: func(ctx context.Context, req *pb.InvokeRequest, opts ...grpc.CallOption) (*pb.InvokeResponse, error) {
			if req.Method != "ListHelmCharts" {
				t.Errorf("expected method 'ListHelmCharts', got %q", req.Method)
			}
			return &pb.InvokeResponse{PayloadJson: `[{"name":"chart1"}]`, Error: ""}, nil
		},
	}
	loader.SetClient(mockClient)
	a.pluginLoaders["helm"] = loader

	result, err := a.InvokePlugin("helm", "ListHelmCharts", "{}")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != `[{"name":"chart1"}]` {
		t.Errorf("expected result '[{\"name\":\"chart1\"}]', got %q", result)
	}
}

func TestInvokePluginErrorResponse(t *testing.T) {
	a := &App{
		pluginLoaders: make(map[string]*plugin.PluginLoader),
		activeContext: "test-context",
		settings: config.Settings{
			KubeconfigPaths: []string{"/tmp/kubeconfig"},
		},
	}

	loader := plugin.NewPluginLoader("helm", "")
	loader.SetStatus(dto.PluginStatusReady)

	mockClient := &mockPluginClient{
		SetClusterContextFn: func(ctx context.Context, req *pb.SetClusterContextRequest, opts ...grpc.CallOption) (*pb.Empty, error) {
			return &pb.Empty{}, nil
		},
		InvokeFn: func(ctx context.Context, req *pb.InvokeRequest, opts ...grpc.CallOption) (*pb.InvokeResponse, error) {
			return &pb.InvokeResponse{PayloadJson: "", Error: "chart not found"}, nil
		},
	}
	loader.SetClient(mockClient)
	a.pluginLoaders["helm"] = loader

	_, err := a.InvokePlugin("helm", "GetHelmChart", `{"name":"notfound"}`)
	if err == nil {
		t.Error("expected error from plugin response")
	}
	if err.Error() != "chart not found" {
		t.Errorf("unexpected error: %v", err)
	}
}

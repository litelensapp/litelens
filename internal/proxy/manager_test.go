package proxy

import (
	"encoding/json"
	"io"
	"os"
	"sync"
	"testing"
	"time"
)

func TestManagerHappyPath(t *testing.T) {
	var mu sync.Mutex
	events := []string{}
	m := NewManager("test-ctx", "sh -c 'echo LITELENS_SETUP_READY'", func(name, msg string) {
		mu.Lock()
		events = append(events, name)
		mu.Unlock()
	})
	m.SetupTimeout(100 * time.Millisecond)

	err := m.Connect()
	if err != nil {
		t.Fatalf("Connect failed: %v", err)
	}

	time.Sleep(200 * time.Millisecond)

	mu.Lock()
	defer mu.Unlock()
	if len(events) == 0 {
		t.Fatal("expected SetupCommandReady event")
	}
	if events[0] != "SetupCommandReady" {
		t.Errorf("expected SetupCommandReady, got %s", events[0])
	}
}

func TestManagerTimeout(t *testing.T) {
	var mu sync.Mutex
	events := []string{}
	m := NewManager("test-ctx", "sh -c 'sleep 10'", func(name, msg string) {
		mu.Lock()
		events = append(events, name)
		mu.Unlock()
	})
	m.SetupTimeout(100 * time.Millisecond)

	err := m.Connect()
	if err != nil {
		t.Fatalf("Connect failed: %v", err)
	}

	time.Sleep(300 * time.Millisecond)

	mu.Lock()
	defer mu.Unlock()
	found := false
	for _, e := range events {
		if e == "SetupCommandDegraded" {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("expected SetupCommandDegraded event, got %v", events)
	}
}

func TestManagerStopDuringStarting(t *testing.T) {
	var mu sync.Mutex
	events := []string{}
	m := NewManager("test-ctx", "sh -c 'sleep 10'", func(name, msg string) {
		mu.Lock()
		events = append(events, name)
		mu.Unlock()
	})
	m.SetupTimeout(5 * time.Second)

	err := m.Connect()
	if err != nil {
		t.Fatalf("Connect failed: %v", err)
	}

	time.Sleep(10 * time.Millisecond)
	m.Stop()

	time.Sleep(100 * time.Millisecond)

	mu.Lock()
	defer mu.Unlock()
	found := false
	for _, e := range events {
		if e == "SetupCommandDegraded" || e == "SetupCommandReady" || e == "SetupCommandCrashed" {
			found = true
			break
		}
	}
	if found {
		t.Errorf("should not emit events after Stop during Starting, got %v", events)
	}
}

func TestManagerStopDuringReady(t *testing.T) {
	var mu sync.Mutex
	events := []string{}
	m := NewManager("test-ctx", "sh -c 'echo LITELENS_SETUP_READY; sleep 10'", func(name, msg string) {
		mu.Lock()
		events = append(events, name)
		mu.Unlock()
	})
	m.SetupTimeout(5 * time.Second)

	err := m.Connect()
	if err != nil {
		t.Fatalf("Connect failed: %v", err)
	}

	time.Sleep(200 * time.Millisecond)
	m.Stop()

	time.Sleep(100 * time.Millisecond)

	mu.Lock()
	defer mu.Unlock()
	readyFound := false
	for _, e := range events {
		if e == "SetupCommandReady" {
			readyFound = true
			break
		}
	}
	if !readyFound {
		t.Errorf("expected SetupCommandReady before Stop, got %v", events)
	}
}

func TestManagerReconnectReusesReady(t *testing.T) {
	var mu sync.Mutex
	events := []string{}
	m := NewManager("test-ctx", "sh -c 'echo LITELENS_SETUP_READY; sleep 10'", func(name, msg string) {
		mu.Lock()
		events = append(events, name)
		mu.Unlock()
	})
	m.SetupTimeout(5 * time.Second)

	err := m.Connect()
	if err != nil {
		t.Fatalf("first Connect failed: %v", err)
	}

	time.Sleep(200 * time.Millisecond)

	mu.Lock()
	readyCount := 0
	for _, e := range events {
		if e == "SetupCommandReady" {
			readyCount++
		}
	}
	mu.Unlock()

	err = m.Connect()
	if err != nil {
		t.Fatalf("second Connect failed: %v", err)
	}

	time.Sleep(100 * time.Millisecond)

	mu.Lock()
	readyCountAfter := 0
	for _, e := range events {
		if e == "SetupCommandReady" {
			readyCountAfter++
		}
	}
	mu.Unlock()

	if readyCount != 1 || readyCountAfter != 1 {
		t.Errorf("expected exactly one SetupCommandReady event even after second Connect, got %d then %d", readyCount, readyCountAfter)
	}
}

func TestScanExactLine(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		target   string
		expected bool
	}{
		{
			name:     "exact match",
			input:    "LITELENS_SETUP_READY\n",
			target:   "LITELENS_SETUP_READY",
			expected: true,
		},
		{
			name:     "with whitespace",
			input:    "  LITELENS_SETUP_READY  \n",
			target:   "LITELENS_SETUP_READY",
			expected: true,
		},
		{
			name:     "no match",
			input:    "other line\n",
			target:   "LITELENS_SETUP_READY",
			expected: false,
		},
		{
			name:     "partial match",
			input:    "LITELENS_SETUP_READY_NOT\n",
			target:   "LITELENS_SETUP_READY",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r, w, err := os.Pipe()
			if err != nil {
				t.Fatalf("failed to create pipe: %v", err)
			}
			defer r.Close()

			notifyChan := make(chan struct{})

			go scanExactLine(r, tt.target, notifyChan, "test-context")

			_, err = io.WriteString(w, tt.input)
			if err != nil {
				t.Fatalf("failed to write: %v", err)
			}
			w.Close()

			if tt.expected {
				select {
				case <-notifyChan:
				case <-time.After(1 * time.Second):
					t.Fatal("expected notification but timed out")
				}
			} else {
				select {
				case <-notifyChan:
					t.Fatal("got unexpected notification")
				case <-time.After(100 * time.Millisecond):
				}
			}
		})
	}
}

func TestManagerStopDuringIdle(t *testing.T) {
	var mu sync.Mutex
	events := []string{}
	m := NewManager("test-ctx", "sh -c 'echo LITELENS_SETUP_READY'", func(name, msg string) {
		mu.Lock()
		events = append(events, name)
		mu.Unlock()
	})

	m.Stop()

	mu.Lock()
	defer mu.Unlock()
	if len(events) > 0 {
		t.Errorf("expected no events when stopping during Idle, got %v", events)
	}
}

func TestManagerCrashAfterReady(t *testing.T) {
	var mu sync.Mutex
	events := []string{}
	m := NewManager("test-ctx", "sh -c 'echo LITELENS_SETUP_READY; sleep 1; exit 1'", func(name, msg string) {
		mu.Lock()
		events = append(events, name)
		mu.Unlock()
	})
	m.SetupTimeout(5 * time.Second)

	err := m.Connect()
	if err != nil {
		t.Fatalf("Connect failed: %v", err)
	}

	time.Sleep(500 * time.Millisecond)

	mu.Lock()
	readyFound := false
	for _, e := range events {
		if e == "SetupCommandReady" {
			readyFound = true
			break
		}
	}
	mu.Unlock()

	if !readyFound {
		t.Fatal("expected SetupCommandReady event before process crash")
	}

	time.Sleep(1500 * time.Millisecond)

	mu.Lock()
	crashFound := false
	for _, e := range events {
		if e == "SetupCommandCrashed" {
			crashFound = true
			break
		}
	}
	mu.Unlock()

	if !crashFound {
		t.Errorf("expected SetupCommandCrashed event after process death, got %v", events)
	}
}

func TestManagerConcurrentConnect(t *testing.T) {
	var mu sync.Mutex
	events := []string{}
	m := NewManager("test-ctx", "sh -c 'echo LITELENS_SETUP_READY; sleep 10'", func(name, msg string) {
		mu.Lock()
		events = append(events, name)
		mu.Unlock()
	})
	m.SetupTimeout(5 * time.Second)

	err1 := m.Connect()
	err2 := m.Connect()

	if err1 != nil {
		t.Fatalf("first Connect failed: %v", err1)
	}
	if err2 != nil {
		t.Fatalf("second Connect should succeed (no-op): %v", err2)
	}

	time.Sleep(300 * time.Millisecond)

	mu.Lock()
	readyCount := 0
	for _, e := range events {
		if e == "SetupCommandReady" {
			readyCount++
		}
	}
	mu.Unlock()

	if readyCount != 1 {
		t.Errorf("expected exactly one SetupCommandReady event with concurrent Connects, got %d", readyCount)
	}
}

func TestClusterProxyRoundTrip(t *testing.T) {
	type ClusterProxy struct {
		HttpProxy    string `json:"httpProxy"`
		HttpsProxy   string `json:"httpsProxy"`
		SetupCommand string `json:"setupCommand"`
	}

	original := ClusterProxy{
		HttpProxy:    "http://proxy.example.com:8080",
		HttpsProxy:   "https://proxy.example.com:8443",
		SetupCommand: "ssm-proxy start",
	}

	data, err := json.Marshal(original)
	if err != nil {
		t.Fatalf("marshal failed: %v", err)
	}

	var unmarshaled ClusterProxy
	err = json.Unmarshal(data, &unmarshaled)
	if err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}

	if unmarshaled.HttpProxy != original.HttpProxy {
		t.Errorf("httpProxy mismatch: %v vs %v", unmarshaled.HttpProxy, original.HttpProxy)
	}
	if unmarshaled.HttpsProxy != original.HttpsProxy {
		t.Errorf("httpsProxy mismatch: %v vs %v", unmarshaled.HttpsProxy, original.HttpsProxy)
	}
	if unmarshaled.SetupCommand != original.SetupCommand {
		t.Errorf("setupCommand mismatch: %v vs %v", unmarshaled.SetupCommand, original.SetupCommand)
	}
}

// TestManagerWaitBlocksUntilReady guards against the bug where a caller
// (App.Connect) started the setup command but never actually paused for it —
// it raced ahead to connect while e.g. an SSO browser flow the command opens
// was still in progress. Wait() must block until the launch settles.
func TestManagerWaitBlocksUntilReady(t *testing.T) {
	m := NewManager("test-ctx", "sleep 0.2 && echo LITELENS_SETUP_READY", func(name, msg string) {})
	m.SetupTimeout(5 * time.Second)

	if err := m.Connect(); err != nil {
		t.Fatalf("Connect failed: %v", err)
	}

	start := time.Now()
	select {
	case <-m.Wait():
	case <-time.After(2 * time.Second):
		t.Fatal("Wait() did not unblock in time")
	}
	elapsed := time.Since(start)

	if elapsed < 150*time.Millisecond {
		t.Errorf("Wait() returned too early (%v); it should have paused for the command to signal readiness", elapsed)
	}

	// A subsequent Wait() call on an already-settled manager must return
	// immediately, not re-block on a stale/new channel.
	select {
	case <-m.Wait():
	default:
		t.Error("expected Wait() to be immediately ready after the manager settled")
	}
}

// TestManagerWaitBlocksUntilTimeout ensures Wait() also pauses for the full
// fail-open timeout path, not just the happy path.
func TestManagerWaitBlocksUntilTimeout(t *testing.T) {
	m := NewManager("test-ctx", "sleep 5", func(name, msg string) {})
	m.SetupTimeout(150 * time.Millisecond)

	if err := m.Connect(); err != nil {
		t.Fatalf("Connect failed: %v", err)
	}

	start := time.Now()
	select {
	case <-m.Wait():
	case <-time.After(2 * time.Second):
		t.Fatal("Wait() did not unblock in time")
	}
	elapsed := time.Since(start)

	if elapsed < 100*time.Millisecond {
		t.Errorf("Wait() returned too early (%v); it should have paused through the setup timeout", elapsed)
	}
}

// TestManagerWaitImmediateWhenNeverConnected ensures Wait() never blocks a
// caller when Connect() was never invoked (e.g. no setup command configured).
func TestManagerWaitImmediateWhenNeverConnected(t *testing.T) {
	m := NewManager("test-ctx", "", func(name, msg string) {})

	select {
	case <-m.Wait():
	default:
		t.Error("expected Wait() to be immediately ready when Connect() was never called")
	}
}

package plugin

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sync"
	"time"

	"github.com/litelensapp/litelens/packages/core/dto"
	"github.com/litelensapp/litelens/packages/core/pb"
	"google.golang.org/grpc"
)

// PluginLoader manages a single plugin instance lifecycle
type PluginLoader struct {
	id               string
	binaryPath       string
	lockFilePath     string
	mu               sync.Mutex
	status           dto.PluginStatus
	progress         int // 0-100 download progress
	conn             *grpc.ClientConn
	client           pb.PluginClient
	processCmd       *exec.Cmd
	cancelHealthLoop context.CancelFunc
	lastError        string
	backendAddr      string // HTTP backend address (127.0.0.1:<port>), if provided by plugin handshake
	onRestart        func(id string, backendAddr string) // callback to emit plugin:backendRestarted event
}

// NewPluginLoader creates a new loader for a plugin. The lock file lives
// alongside the binary (i.e. under the caller's configured plugins root),
// not a hardcoded default — otherwise switching to a custom plugins
// directory would leave lock-file lookups pointed at the old location,
// letting Launch() reuse a stale process from a completely different
// directory (Launch only verifies PID-liveness + a gRPC health check, never
// that the process was spawned from the currently configured binary).
func NewPluginLoader(id string, binaryPath string) *PluginLoader {
	lockDir := filepath.Dir(binaryPath)
	lockFile := filepath.Join(lockDir, id+".lock")
	return &PluginLoader{
		id:           id,
		binaryPath:   binaryPath,
		lockFilePath: lockFile,
		status:       dto.PluginStatusNotInstalled,
	}
}

// Launch starts or reuses a plugin instance with an optional kubeconfig path.
func (pl *PluginLoader) Launch(ctx context.Context, kubeconfigPath string) error {
	pl.mu.Lock()
	defer pl.mu.Unlock()

	// Check if lock file exists with a live process
	if lockData, err := pl.readLockFile(); err == nil && lockData != nil {
		if isProcessAlive(lockData.PID) {
			// Try health check
			if pl.dialAndHealthCheck(ctx, lockData.Port) {
				pl.status = dto.PluginStatusReady
				pl.startHealthLoop()
				return nil
			}
		}
		// Stale lock, delete it
		_ = os.Remove(pl.lockFilePath)
	}

	// Spawn plugin process with kubeconfig argument
	args := []string{}
	if kubeconfigPath != "" {
		args = append(args, "-kubeconfig", kubeconfigPath)
	}
	cmd := exec.CommandContext(ctx, pl.binaryPath, args...)

	// Log the command being launched for troubleshooting
	fmt.Printf("launching plugin %s with args %v\n", pl.binaryPath, cmd.Args)

	stdoutPipe, err := cmd.StdoutPipe()
	if err != nil {
		pl.status = dto.PluginStatusCrashed
		pl.lastError = fmt.Sprintf("stdout pipe: %v", err)
		return fmt.Errorf("plugin launch failed: %w", err)
	}

	if err := cmd.Start(); err != nil {
		pl.status = dto.PluginStatusCrashed
		pl.lastError = fmt.Sprintf("start process: %v", err)
		return fmt.Errorf("plugin start failed: %w", err)
	}

	pl.processCmd = cmd

	// Read handshake within 5s timeout
	readCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	handshakeCh := make(chan map[string]interface{}, 1)
	errCh := make(chan error, 1)

	go func() {
		reader := bufio.NewReader(stdoutPipe)
		line, err := reader.ReadString('\n')
		if err != nil {
			errCh <- fmt.Errorf("read stdout: %w", err)
			return
		}
		if len(line) == 0 {
			errCh <- fmt.Errorf("no output from plugin")
			return
		}
		var handshake map[string]interface{}
		if err := json.Unmarshal([]byte(line), &handshake); err != nil {
			errCh <- fmt.Errorf("parse handshake: %w", err)
			return
		}
		handshakeCh <- handshake
	}()

	select {
	case <-readCtx.Done():
		_ = cmd.Process.Kill()
		pl.status = dto.PluginStatusCrashed
		pl.lastError = "handshake timeout (5s)"
		return fmt.Errorf("plugin handshake timeout")
	case err := <-errCh:
		_ = cmd.Process.Kill()
		pl.status = dto.PluginStatusCrashed
		pl.lastError = err.Error()
		return fmt.Errorf("read handshake: %w", err)
	case handshake := <-handshakeCh:
		if err := pl.validateHandshake(handshake); err != nil {
			_ = cmd.Process.Kill()
			pl.status = dto.PluginStatusCrashed
			pl.lastError = err.Error()
			return err
		}

		// Extract gRPC port (required)
		port := int(handshake["grpcPort"].(float64))

		// Extract HTTP backend address if provided by plugin (optional, Phase 5+)
		backendAddr := ""
		if httpAddrVal, ok := handshake["httpAddr"]; ok {
			if httpAddrStr, ok := httpAddrVal.(string); ok && httpAddrStr != "" {
				backendAddr = httpAddrStr
			}
		}

		// Write lock file
		if err := pl.writeLockFile(cmd.Process.Pid, port); err != nil {
			_ = cmd.Process.Kill()
			pl.status = dto.PluginStatusCrashed
			pl.lastError = fmt.Sprintf("write lock file: %v", err)
			return err
		}

		// Dial gRPC
		if err := pl.dialGRPC(ctx, fmt.Sprintf("127.0.0.1:%d", port)); err != nil {
			_ = cmd.Process.Kill()
			pl.status = dto.PluginStatusCrashed
			pl.lastError = fmt.Sprintf("dial grpc: %v", err)
			return err
		}

		// Store backend address if available. pl.mu is already held for the
		// duration of Launch() (see the defer at the top) — locking again here
		// would deadlock since sync.Mutex is not reentrant.
		pl.backendAddr = backendAddr

		pl.status = dto.PluginStatusReady
		pl.startHealthLoop()
		return nil
	}
}

// validateHandshake checks the handshake JSON for required fields and valid port.
func (pl *PluginLoader) validateHandshake(handshake map[string]interface{}) error {
	if handshake["type"] != "READY" {
		return fmt.Errorf("invalid handshake type: %v", handshake["type"])
	}
	if _, ok := handshake["grpcPort"]; !ok {
		return fmt.Errorf("missing grpcPort in handshake")
	}
	port, ok := handshake["grpcPort"].(float64)
	if !ok {
		return fmt.Errorf("invalid grpcPort type")
	}
	portInt := int(port)
	if portInt < 1 || portInt > 65535 {
		return fmt.Errorf("grpcPort out of range: %d", portInt)
	}
	return nil
}

// dialAndHealthCheck attempts to dial and check health. On success it stores
// the connection/client on pl so callers (e.g. Launch's reuse-existing-process
// path) end up with a usable client, not just a bool.
func (pl *PluginLoader) dialAndHealthCheck(ctx context.Context, port int) bool {
	addr := fmt.Sprintf("127.0.0.1:%d", port)
	conn, err := grpc.DialContext(ctx, addr, grpc.WithInsecure())
	if err != nil {
		return false
	}
	client := pb.NewPluginClient(conn)

	checkCtx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()

	if _, err := client.GetCapabilities(checkCtx, &pb.Empty{}); err != nil {
		conn.Close()
		return false
	}

	pl.conn = conn
	pl.client = client
	return true
}

// dialGRPC establishes a gRPC connection
func (pl *PluginLoader) dialGRPC(ctx context.Context, addr string) error {
	conn, err := grpc.DialContext(ctx, addr, grpc.WithInsecure())
	if err != nil {
		return err
	}
	pl.conn = conn
	pl.client = pb.NewPluginClient(conn)
	return nil
}

// readLockFile reads and parses the lock file
func (pl *PluginLoader) readLockFile() (*dto.PluginLockFile, error) {
	data, err := os.ReadFile(pl.lockFilePath)
	if err != nil {
		return nil, err
	}
	var lockFile dto.PluginLockFile
	if err := json.Unmarshal(data, &lockFile); err != nil {
		return nil, err
	}
	return &lockFile, nil
}

// writeLockFile writes the lock file
func (pl *PluginLoader) writeLockFile(pid, port int) error {
	lockDir := filepath.Dir(pl.lockFilePath)
	if err := os.MkdirAll(lockDir, 0700); err != nil {
		return err
	}
	lockFile := dto.PluginLockFile{
		PID:       pid,
		Port:      port,
		Timestamp: time.Now().Format(time.RFC3339),
		Version:   "v1",
	}
	data, _ := json.MarshalIndent(lockFile, "", "  ")
	return os.WriteFile(pl.lockFilePath, data, 0600)
}

// startHealthLoop starts the background health check goroutine.
// Must be called with pl.mu already held by the caller (see Launch) — it does
// not lock internally, since re-locking here would deadlock against the
// caller's held lock (sync.Mutex is not reentrant).
func (pl *PluginLoader) startHealthLoop() {
	// Cancel any prior health loop before starting a new one.
	if pl.cancelHealthLoop != nil {
		pl.cancelHealthLoop()
	}

	ctx, cancel := context.WithCancel(context.Background())
	pl.cancelHealthLoop = cancel

	go func() {
		failures := 0
		ticker := time.NewTicker(30 * time.Second)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				// Snapshot pl.client under lock to avoid data race
				pl.mu.Lock()
				client := pl.client
				pl.mu.Unlock()

				if client == nil {
					return
				}

				checkCtx, checkCancel := context.WithTimeout(ctx, 2*time.Second)
				_, err := client.GetCapabilities(checkCtx, &pb.Empty{})
				checkCancel()

				if err != nil {
					failures++
					if failures >= 3 {
						pl.mu.Lock()
						pl.status = dto.PluginStatusCrashed
						pl.lastError = fmt.Sprintf("health check failed: %v", err)
						backendAddr := pl.backendAddr
						onRestart := pl.onRestart
						pl.mu.Unlock()

						// Kill process and delete lock
						if pl.processCmd != nil && pl.processCmd.Process != nil {
							_ = pl.processCmd.Process.Kill()
						}
						_ = os.Remove(pl.lockFilePath)
						_ = pl.conn.Close()

						// Emit restart event if callback is set
						if onRestart != nil {
							onRestart(pl.id, backendAddr)
						}

						return
					}
				} else {
					failures = 0
				}
			}
		}
	}()
}

// Status returns the current plugin status (thread-safe)
func (pl *PluginLoader) Status() dto.PluginStatus {
	pl.mu.Lock()
	defer pl.mu.Unlock()
	return pl.status
}

// LastError returns the last recorded error message (thread-safe)
func (pl *PluginLoader) LastError() string {
	pl.mu.Lock()
	defer pl.mu.Unlock()
	return pl.lastError
}

// Progress returns the current download progress as a percentage (0-100) (thread-safe)
func (pl *PluginLoader) Progress() int {
	pl.mu.Lock()
	defer pl.mu.Unlock()
	return pl.progress
}

// SetProgress sets the download progress as a percentage (0-100) (thread-safe)
func (pl *PluginLoader) SetProgress(pct int) {
	pl.mu.Lock()
	defer pl.mu.Unlock()
	pl.progress = pct
}

// SetStatus sets the plugin status (thread-safe); resets progress to 0 except for READY status
func (pl *PluginLoader) SetStatus(status dto.PluginStatus) {
	pl.mu.Lock()
	defer pl.mu.Unlock()
	pl.status = status
	pl.lastError = ""
	// Set progress to 100 when status is READY, otherwise reset to 0
	if status == dto.PluginStatusReady {
		pl.progress = 100
	} else {
		pl.progress = 0
	}
}

// SetStatusWithError sets the plugin status with an error message (thread-safe); resets progress to 0
func (pl *PluginLoader) SetStatusWithError(status dto.PluginStatus, errMsg string) {
	pl.mu.Lock()
	defer pl.mu.Unlock()
	pl.status = status
	pl.lastError = errMsg
	pl.progress = 0
}

// BinaryPath returns the path to the plugin binary
func (pl *PluginLoader) BinaryPath() string {
	return pl.binaryPath
}

// SetBinaryPath sets the path to the plugin binary (thread-safe).
// Used when the real binary path is determined after loader creation.
func (pl *PluginLoader) SetBinaryPath(binaryPath string) {
	pl.mu.Lock()
	defer pl.mu.Unlock()
	pl.binaryPath = binaryPath
	// Update lock file path to match the new binary location
	pl.lockFilePath = filepath.Join(filepath.Dir(binaryPath), pl.id+".lock")
}

// GetClient returns the gRPC client (requires status check first)
func (pl *PluginLoader) GetClient() pb.PluginClient {
	pl.mu.Lock()
	defer pl.mu.Unlock()
	return pl.client
}

// GetBackendAddr returns the HTTP backend address (127.0.0.1:<port>), or "" if not available
func (pl *PluginLoader) GetBackendAddr() string {
	pl.mu.Lock()
	defer pl.mu.Unlock()
	return pl.backendAddr
}

// SetOnRestart sets a callback to be invoked when the plugin restarts. The
// callback runs from the health-loop goroutine with pl.mu NOT held (it is
// read and released before invocation, see startHealthLoop) — but it must
// still stay cheap and non-blocking, since it runs synchronously on that
// goroutine before the health loop exits.
func (pl *PluginLoader) SetOnRestart(callback func(id string, backendAddr string)) {
	pl.mu.Lock()
	defer pl.mu.Unlock()
	pl.onRestart = callback
}

// SetClient sets the gRPC client directly (used for testing)
func (pl *PluginLoader) SetClient(client pb.PluginClient) {
	pl.mu.Lock()
	defer pl.mu.Unlock()
	pl.client = client
}

// Shutdown cleanly shuts down the plugin
func (pl *PluginLoader) Shutdown() error {
	pl.mu.Lock()
	defer pl.mu.Unlock()

	if pl.cancelHealthLoop != nil {
		pl.cancelHealthLoop()
	}

	if pl.processCmd != nil && pl.processCmd.Process != nil {
		_ = pl.processCmd.Process.Kill()
		_ = pl.processCmd.Wait()
	}

	if pl.conn != nil {
		_ = pl.conn.Close()
	}

	_ = os.Remove(pl.lockFilePath)
	return nil
}

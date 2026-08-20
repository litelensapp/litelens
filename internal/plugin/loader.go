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
)

// PluginLoader manages a single plugin instance lifecycle
type PluginLoader struct {
	id           string
	binaryPath   string
	lockFilePath string
	mu           sync.Mutex
	status       dto.PluginStatus
	progress     int // 0-100 download progress
	pid          int // plugin subprocess PID, used for on-demand liveness checks
	processCmd   *exec.Cmd
	lastError    string
	hostGRPCPort int // host's gRPC port for cluster context watch (0 = not set)
}

// NewPluginLoader creates a new loader for a plugin. The lock file lives
// alongside the binary (i.e. under the caller's configured plugins root),
// not a hardcoded default — otherwise switching to a custom plugins
// directory would leave lock-file lookups pointed at the old location,
// letting Launch() reuse a stale process from a completely different
// directory (Launch only verifies PID-liveness, never that the process was
// spawned from the currently configured binary).
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

	// Check if lock file exists with a live process. Liveness is PID-only —
	// there is no network health check anymore; a process that's alive but
	// unresponsive is caught later by the frontend's on-demand retry path.
	if lockData, err := pl.readLockFile(); err == nil && lockData != nil {
		if isProcessAlive(lockData.PID) {
			pl.pid = lockData.PID
			pl.status = dto.PluginStatusReady
			return nil
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

	// Set environment variable for host gRPC port. pl.mu is already held for
	// the duration of Launch() (see the defer at the top) — locking again here
	// would deadlock since sync.Mutex is not reentrant.
	hostGRPCPort := pl.hostGRPCPort
	if hostGRPCPort > 0 {
		cmd.Env = append(os.Environ(), fmt.Sprintf("LITELENS_HOST_GRPC_PORT=%d", hostGRPCPort))
	}

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

		// Extract HTTP port (required)
		httpPort := int(handshake["httpPort"].(float64))

		// Write lock file
		if err := pl.writeLockFile(cmd.Process.Pid, httpPort); err != nil {
			_ = cmd.Process.Kill()
			pl.status = dto.PluginStatusCrashed
			pl.lastError = fmt.Sprintf("write lock file: %v", err)
			return err
		}

		// pl.mu is already held for the duration of Launch() (see the defer at
		// the top) — locking again here would deadlock since sync.Mutex is not
		// reentrant.
		pl.pid = cmd.Process.Pid

		pl.status = dto.PluginStatusReady
		return nil
	}
}

// validateHandshake checks the handshake JSON for required fields and valid port.
func (pl *PluginLoader) validateHandshake(handshake map[string]interface{}) error {
	if handshake["type"] != "READY" {
		return fmt.Errorf("invalid handshake type: %v", handshake["type"])
	}
	if _, ok := handshake["httpPort"]; !ok {
		return fmt.Errorf("missing httpPort in handshake")
	}
	port, ok := handshake["httpPort"].(float64)
	if !ok {
		return fmt.Errorf("invalid httpPort type")
	}
	portInt := int(port)
	if portInt < 1 || portInt > 65535 {
		return fmt.Errorf("httpPort out of range: %d", portInt)
	}
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

// IsAlive reports whether the plugin's subprocess PID is still alive. This is
// the sole liveness signal used for on-demand relaunch decisions — there is
// no network health check; a process that's alive but unresponsive is not
// detected here.
func (pl *PluginLoader) IsAlive() bool {
	pl.mu.Lock()
	defer pl.mu.Unlock()
	return pl.pid != 0 && isProcessAlive(pl.pid)
}

// HTTPPort reads and returns the plugin's HTTP backend port from the lock file.
// Returns an error if the lock file cannot be read or is invalid.
func (pl *PluginLoader) HTTPPort() (int, error) {
	pl.mu.Lock()
	defer pl.mu.Unlock()
	lockData, err := pl.readLockFile()
	if err != nil {
		return 0, err
	}
	return lockData.Port, nil
}

// SetHostGRPCPort sets the host's gRPC port for cluster context watch (thread-safe)
func (pl *PluginLoader) SetHostGRPCPort(port int) {
	pl.mu.Lock()
	defer pl.mu.Unlock()
	pl.hostGRPCPort = port
}

// Shutdown cleanly shuts down the plugin
func (pl *PluginLoader) Shutdown() error {
	pl.mu.Lock()
	defer pl.mu.Unlock()

	if pl.processCmd != nil && pl.processCmd.Process != nil {
		_ = pl.processCmd.Process.Kill()
		_ = pl.processCmd.Wait()
	}

	_ = os.Remove(pl.lockFilePath)
	return nil
}

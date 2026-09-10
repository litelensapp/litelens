package proxy

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"log"
	"os/exec"
	"strings"
	"sync"
	"time"
)

type State int

const (
	Idle State = iota
	Starting
	Ready
	Degraded
	Stopping
)

type Manager struct {
	contextName string
	command     string
	emitEvent   func(eventName, message string)

	mu          sync.Mutex
	state       State
	launchSeq   int64
	cmd         *exec.Cmd
	cancelFunc  context.CancelFunc
	settledChan chan struct{}

	setupTimeout time.Duration
}

func NewManager(contextName, command string, emitEvent func(eventName, message string)) *Manager {
	closed := make(chan struct{})
	close(closed)
	return &Manager{
		contextName:  contextName,
		command:      command,
		emitEvent:    emitEvent,
		state:        Idle,
		settledChan:  closed,
		setupTimeout: 5 * time.Minute,
	}
}

func (m *Manager) SetupTimeout(d time.Duration) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.setupTimeout = d
}

func (m *Manager) Command() string {
	return m.command
}

// Wait returns a channel that closes once the in-flight (or most recently
// completed) launch has settled into Idle, Ready, or Degraded. If Connect
// hasn't been called since the manager was created, or since it last
// returned to Idle, the channel is already closed. Callers that want
// Connect() to pause until the setup command signals readiness (e.g. an SSO
// browser flow the command opens) should call Connect() followed by
// <-mgr.Wait().
func (m *Manager) Wait() <-chan struct{} {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.settledChan
}

func (m *Manager) Connect() error {
	m.mu.Lock()
	if m.state != Idle {
		m.mu.Unlock()
		return nil
	}
	m.state = Starting
	m.launchSeq++
	seq := m.launchSeq
	ctx, cancel := context.WithCancel(context.Background())
	m.cancelFunc = cancel
	settled := make(chan struct{})
	m.settledChan = settled
	m.mu.Unlock()

	go m.doLaunch(ctx, seq, settled)
	return nil
}

func (m *Manager) doLaunch(ctx context.Context, seq int64, settled chan struct{}) {
	defer close(settled)

	readyMarkerChan := make(chan struct{})

	m.mu.Lock()
	if m.launchSeq != seq {
		m.mu.Unlock()
		return
	}
	m.mu.Unlock()

	cmd := exec.CommandContext(ctx, "sh", "-c", m.command)
	setProcessGroup(cmd)
	stdout, _ := cmd.StdoutPipe()
	stderr, _ := cmd.StderrPipe()

	if err := cmd.Start(); err != nil {
		m.transitionTo(Degraded, seq)
		m.emitEvent("SetupCommandDegraded", fmt.Sprintf("failed to start setup command: %v", err))
		return
	}

	m.mu.Lock()
	m.cmd = cmd
	m.mu.Unlock()

	go scanExactLine(stdout, "LITELENS_SETUP_READY", readyMarkerChan)
	go logLines(stderr, m.contextName)

	m.mu.Lock()
	timeoutDuration := m.setupTimeout
	m.mu.Unlock()

	select {
	case <-ctx.Done():
		killProcessGroup(cmd)
		m.transitionTo(Idle, seq)
	case <-readyMarkerChan:
		if m.transitionIfStillStarting(Ready, seq) {
			m.emitEvent("SetupCommandReady", "proxy setup command ready")
			go m.watchProcessDeath(cmd, seq)
		}
	case <-time.After(timeoutDuration):
		killProcessGroup(cmd)
		m.transitionTo(Degraded, seq)
		m.emitEvent("SetupCommandDegraded", "setup command timed out after 5m; proceeding without working proxy")
	}
}

func (m *Manager) watchProcessDeath(cmd *exec.Cmd, seq int64) {
	cmd.Wait()
	if m.transitionIfStillInState(Ready, Degraded, seq) {
		m.emitEvent("SetupCommandCrashed", "setup command process exited unexpectedly after reaching ready state")
	}
}

func (m *Manager) Stop() {
	m.mu.Lock()
	switch m.state {
	case Idle:
		m.mu.Unlock()
		return
	case Starting:
		cancel := m.cancelFunc
		m.state = Stopping
		m.mu.Unlock()
		cancel() // doLaunch's ctx.Done() branch kills the process and transitions to Idle
	case Ready, Degraded:
		cmd := m.cmd
		m.state = Idle
		m.mu.Unlock()
		killProcessGroup(cmd)
	case Stopping:
		m.mu.Unlock()
	}
}

func (m *Manager) transitionTo(newState State, seq int64) bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.launchSeq != seq {
		return false
	}
	m.state = newState
	return true
}

func (m *Manager) transitionIfStillStarting(newState State, seq int64) bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.launchSeq != seq || m.state != Starting {
		return false
	}
	m.state = newState
	return true
}

func (m *Manager) transitionIfStillInState(from, to State, seq int64) bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.launchSeq != seq || m.state != from {
		return false
	}
	m.state = to
	return true
}

func scanExactLine(source io.Reader, target string, notifyChan chan struct{}) {
	scanner := bufio.NewScanner(source)
	for scanner.Scan() {
		if strings.TrimSpace(scanner.Text()) == target {
			close(notifyChan)
			return
		}
	}
	if err := scanner.Err(); err != nil {
		log.Printf("error reading setup command stdout: %v", err)
	}
}

func logLines(source io.Reader, contextName string) {
	scanner := bufio.NewScanner(source)
	for scanner.Scan() {
		log.Printf("[setup-command:%s] %s", contextName, scanner.Text())
	}
	if err := scanner.Err(); err != nil {
		log.Printf("error reading setup command stderr for %s: %v", contextName, err)
	}
}

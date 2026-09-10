package proxy

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"log"
	"net"
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

func (s State) String() string {
	switch s {
	case Idle:
		return "idle"
	case Starting:
		return "starting"
	case Ready:
		return "ready"
	case Degraded:
		return "degraded"
	case Stopping:
		return "stopping"
	default:
		return "unknown"
	}
}

// Status is a snapshot of a Manager's current state, for callers (e.g. the
// frontend footer) that want to display it without subscribing to events.
type Status struct {
	State   string
	Message string
}

type Manager struct {
	contextName string
	command     string
	proxyAddr   string // host:port to poll as a readiness fallback; see doLaunch
	emitEvent   func(eventName, message string)

	mu          sync.Mutex
	state       State
	message     string
	launchSeq   int64
	cmd         *exec.Cmd
	cancelFunc  context.CancelFunc
	settledChan chan struct{}

	setupTimeout time.Duration
}

// NewManager creates a Manager for a setup command. proxyAddr, if non-empty,
// is the host:port of the proxy the command is expected to stand up (i.e.
// this cluster's configured httpProxy/httpsProxy address) — it's polled as a
// readiness fallback alongside the LITELENS_SETUP_READY stdout marker, since
// not every setup command can be made to print a marker (e.g. a script whose
// last step execs a long-running foreground tunnel like `aws ssm
// start-session`, which never returns to reach a later echo). Pass "" to
// rely on the stdout marker alone.
func NewManager(contextName, command, proxyAddr string, emitEvent func(eventName, message string)) *Manager {
	closed := make(chan struct{})
	close(closed)
	return &Manager{
		contextName:  contextName,
		command:      command,
		proxyAddr:    proxyAddr,
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

func (m *Manager) ProxyAddr() string {
	return m.proxyAddr
}

// Status returns a snapshot of the manager's current state and last status
// message, for callers that want to display it without subscribing to events.
func (m *Manager) Status() Status {
	m.mu.Lock()
	defer m.mu.Unlock()
	return Status{State: m.state.String(), Message: m.message}
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
		msg := fmt.Sprintf("failed to start setup command: %v", err)
		m.transitionTo(Degraded, seq, msg)
		m.emitEvent("SetupCommandDegraded", msg)
		return
	}

	m.mu.Lock()
	m.cmd = cmd
	m.mu.Unlock()

	go scanExactLine(stdout, "LITELENS_SETUP_READY", readyMarkerChan, m.contextName)
	go logLines(stderr, m.contextName)

	portReadyChan := make(chan struct{})
	var pollStop chan struct{}
	if m.proxyAddr != "" {
		pollStop = make(chan struct{})
		go pollTCPReady(m.proxyAddr, portReadyChan, pollStop)
	}

	m.mu.Lock()
	timeoutDuration := m.setupTimeout
	m.mu.Unlock()

	select {
	case <-ctx.Done():
		killProcessGroup(cmd)
		log.Printf("[setup-command:%s] proxy server terminated: connect cancelled", m.contextName)
		m.transitionTo(Idle, seq, "")
	case <-readyMarkerChan:
		msg := "proxy setup command ready"
		if m.transitionIfStillStarting(Ready, seq, msg) {
			m.emitEvent("SetupCommandReady", msg)
			go m.watchProcessDeath(cmd, seq)
		}
	case <-portReadyChan:
		msg := "proxy setup command ready (proxy port reachable)"
		if m.transitionIfStillStarting(Ready, seq, msg) {
			m.emitEvent("SetupCommandReady", msg)
			go m.watchProcessDeath(cmd, seq)
		}
	case <-time.After(timeoutDuration):
		killProcessGroup(cmd)
		log.Printf("[setup-command:%s] proxy server terminated: setup command timed out after 5m", m.contextName)
		msg := "setup command timed out after 5m; proceeding without working proxy"
		m.transitionTo(Degraded, seq, msg)
		m.emitEvent("SetupCommandDegraded", msg)
	}

	if pollStop != nil {
		close(pollStop)
	}
}

// pollTCPReady repeatedly dials addr until it accepts a connection, then
// closes notifyChan. Stops early, without closing notifyChan, if stop is
// closed first.
func pollTCPReady(addr string, notifyChan chan struct{}, stop chan struct{}) {
	for {
		conn, err := net.DialTimeout("tcp", addr, 300*time.Millisecond)
		if err == nil {
			conn.Close()
			close(notifyChan)
			return
		}
		select {
		case <-stop:
			return
		case <-time.After(300 * time.Millisecond):
		}
	}
}

func (m *Manager) watchProcessDeath(cmd *exec.Cmd, seq int64) {
	cmd.Wait()
	msg := "setup command process exited unexpectedly after reaching ready state"
	if m.transitionIfStillInState(Ready, Degraded, seq, msg) {
		log.Printf("[setup-command:%s] proxy server terminated: process exited unexpectedly", m.contextName)
		m.emitEvent("SetupCommandCrashed", msg)
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
		m.message = ""
		m.mu.Unlock()
		killProcessGroup(cmd)
		log.Printf("[setup-command:%s] proxy server terminated: disconnected", m.contextName)
	case Stopping:
		m.mu.Unlock()
	}
}

func (m *Manager) transitionTo(newState State, seq int64, message string) bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.launchSeq != seq {
		return false
	}
	m.state = newState
	m.message = message
	return true
}

func (m *Manager) transitionIfStillStarting(newState State, seq int64, message string) bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.launchSeq != seq || m.state != Starting {
		return false
	}
	m.state = newState
	m.message = message
	return true
}

func (m *Manager) transitionIfStillInState(from, to State, seq int64, message string) bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.launchSeq != seq || m.state != from {
		return false
	}
	m.state = to
	m.message = message
	return true
}

func scanExactLine(source io.Reader, target string, notifyChan chan struct{}, contextName string) {
	scanner := bufio.NewScanner(source)
	for scanner.Scan() {
		line := scanner.Text()
		log.Printf("[setup-command:%s] %s", contextName, line)
		if strings.TrimSpace(line) == target {
			close(notifyChan)
			return
		}
	}
	if err := scanner.Err(); err != nil {
		log.Printf("error reading setup command stdout for %s: %v", contextName, err)
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

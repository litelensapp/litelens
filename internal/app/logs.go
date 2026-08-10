package app

import (
	"bufio"
	"context"
	"fmt"
	"log"

	"github.com/wailsapp/wails/v2/pkg/runtime"
	corev1 "k8s.io/api/core/v1"
)

func logKey(ns, pod, container string) string {
	return fmt.Sprintf("log:%s:%s:%s", ns, pod, container)
}

// StreamLogs starts a streaming log tail for the given pod/container.
// Any pre-existing stream with the same key is cancelled first.
func (a *App) StreamLogs(contextName, ns, pod, container string) error {
	key := logKey(ns, pod, container)

	a.streamMu.Lock()
	if cancel, ok := a.logCancels[key]; ok {
		cancel()
	}
	childCtx, cancel := context.WithCancel(a.ctx)
	a.logCancels[key] = cancel
	a.logSeqs[key]++
	seq := a.logSeqs[key]
	a.streamMu.Unlock()

	a.mu.RLock()
	cs := a.clients[contextName]
	a.mu.RUnlock()
	if cs == nil {
		cancel()
		return fmt.Errorf("no client for context %q", contextName)
	}

	req := cs.CoreV1().Pods(ns).GetLogs(pod, &corev1.PodLogOptions{
		Container: container,
		Follow:    true,
	})
	stream, err := req.Stream(childCtx)
	if err != nil {
		cancel()
		return fmt.Errorf("stream logs: %w", err)
	}

	go func() {
		closedKey := "log:closed:" + key
		defer func() {
			stream.Close()
			runtime.EventsEmit(a.ctx, closedKey, nil)
			a.streamMu.Lock()
			if a.logSeqs[key] == seq {
				delete(a.logCancels, key)
			}
			a.streamMu.Unlock()
		}()
		scanner := bufio.NewScanner(stream)
		scanner.Buffer(make([]byte, 256*1024), 1024*1024)
		for scanner.Scan() {
			if !a.isActive(contextName) {
				break
			}
			runtime.EventsEmit(a.ctx, key, scanner.Text())
		}
		if err := scanner.Err(); err != nil {
			log.Printf("app: StreamLogs %s: %v", key, err)
		}
	}()

	return nil
}

// StopLogs cancels the running log stream for the given pod/container.
func (a *App) StopLogs(ns, pod, container string) {
	key := logKey(ns, pod, container)
	a.streamMu.Lock()
	defer a.streamMu.Unlock()
	if cancel, ok := a.logCancels[key]; ok {
		cancel()
		delete(a.logCancels, key)
	}
}

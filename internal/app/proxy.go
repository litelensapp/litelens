package app

import (
	"github.com/litelensapp/litelens/internal/proxy"
	wailsruntime "github.com/wailsapp/wails/v2/pkg/runtime"
)

// ensureProxyManager returns the ProxyManager for contextName, creating one on
// first use. If a manager already exists but was configured with a different
// command (the user edited the setup command in settings), the stale
// manager is stopped and replaced with a fresh one for the new command.
func (a *App) ensureProxyManager(contextName, command string) *proxy.Manager {
	a.proxyManagersMu.RLock()
	if m, exists := a.proxyManagers[contextName]; exists {
		existingCommand := m.Command()
		a.proxyManagersMu.RUnlock()
		if existingCommand == command {
			return m
		}
		m.Stop()
		a.proxyManagersMu.Lock()
		delete(a.proxyManagers, contextName)
		a.proxyManagersMu.Unlock()
	} else {
		a.proxyManagersMu.RUnlock()
	}

	a.proxyManagersMu.Lock()
	defer a.proxyManagersMu.Unlock()

	if m, exists := a.proxyManagers[contextName]; exists {
		if m.Command() == command {
			return m
		}
		m.Stop()
		delete(a.proxyManagers, contextName)
	}

	m := proxy.NewManager(contextName, command, func(eventName, message string) {
		wailsruntime.EventsEmit(a.ctx, eventName, map[string]string{
			"context": contextName,
			"message": message,
		})
	})
	a.proxyManagers[contextName] = m
	return m
}

// stopProxyManager stops and forgets the ProxyManager for contextName, if one
// exists. Called when Connect() switches to a different cluster context.
func (a *App) stopProxyManager(contextName string) {
	a.proxyManagersMu.RLock()
	mgr, exists := a.proxyManagers[contextName]
	a.proxyManagersMu.RUnlock()
	if exists {
		mgr.Stop()
	}
}

// stopAllProxyManagers stops every tracked ProxyManager. Called on app
// shutdown so no setup-command subprocess is left running after the app exits.
func (a *App) stopAllProxyManagers() {
	a.proxyManagersMu.Lock()
	for _, mgr := range a.proxyManagers {
		mgr.Stop()
	}
	a.proxyManagersMu.Unlock()
}

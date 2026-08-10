package app

import (
	"fmt"
	"os"
	"path/filepath"
	goruntime "runtime"
	"strconv"
	"strings"

	"github.com/litelensapp/litelens/internal/config"
	"github.com/litelensapp/litelens/internal/kube"
	"github.com/wailsapp/wails/v2/pkg/runtime"
	"k8s.io/client-go/tools/clientcmd"
)

// OpenAbout emits an event so the frontend opens the About modal.
func (a *App) OpenAbout() {
	runtime.EventsEmit(a.ctx, "menu:open-about", map[string]string{
		"version":      a.version,
		"go":           goruntime.Version(),
		"wails":        config.WailsModuleVersion(),
		"appSizeBytes": strconv.FormatInt(getAppSizeBytes(), 10),
	})
}

// getAppSizeBytes returns the on-disk size of the app in bytes. On macOS
// bundles, it sums the total size of the .app directory; on other platforms
// it returns the size of the executable itself. Returns -1 if the size
// cannot be determined.
func getAppSizeBytes() int64 {
	exe, err := os.Executable()
	if err != nil {
		return -1
	}

	// On macOS, check if running from a .app bundle
	if goruntime.GOOS == "darwin" {
		if idx := strings.Index(exe, ".app/Contents/MacOS/"); idx != -1 {
			// Walk up to the .app directory
			bundleDir := exe[:idx+4] // include the ".app"
			size := int64(0)
			err := filepath.Walk(bundleDir, func(_ string, info os.FileInfo, err error) error {
				if err != nil {
					return err
				}
				if !info.IsDir() {
					size += info.Size()
				}
				return nil
			})
			if err != nil {
				return -1
			}
			return size
		}
	}

	// Fallback: just stat the executable
	info, err := os.Stat(exe)
	if err != nil {
		return -1
	}
	return info.Size()
}

// Quit terminates the application. Called from the native menu and ⌘Q.
func (a *App) Quit() {
	runtime.Quit(a.ctx)
}

// ToggleFullscreen toggles the application window between fullscreen and windowed mode.
func (a *App) ToggleFullscreen() {
	if runtime.WindowIsFullscreen(a.ctx) {
		runtime.WindowUnfullscreen(a.ctx)
	} else {
		runtime.WindowFullscreen(a.ctx)
	}
}

// ExecJS executes JavaScript in the webview. Used by the native Edit menu to
// route clipboard commands through to the WKWebView responder chain.
func (a *App) ExecJS(js string) {
	if a.ctx == nil {
		return
	}
	runtime.WindowExecJS(a.ctx, js)
}

// OpenSettings emits an event so the frontend opens the settings panel.
// Called from the native application menu.
func (a *App) OpenSettings() {
	runtime.EventsEmit(a.ctx, "menu:open-settings")
}

func (a *App) ClipboardGetText() (string, error) {
	return runtime.ClipboardGetText(a.ctx)
}

// SaveSettings persists secrets and other settings. ClusterProxies are managed
// separately via SaveClusterProxy and are preserved here unchanged.
// If PluginsDir changes, rebuilds the plugin loader map to reflect the new directory.
// Marketplace repository URLs are stored exactly as provided, no validation or
// canonicalization — the user is responsible for entering a URL that resolves
// to a GitHub releases API endpoint if they want that source to fetch successfully.
func (a *App) SaveSettings(s config.Settings) error {
	for i := range s.MarketplaceRepositories {
		s.MarketplaceRepositories[i].URL = strings.TrimSpace(s.MarketplaceRepositories[i].URL)
	}

	a.mu.RLock()
	oldPluginsDir := a.settings.PluginsDir
	a.mu.RUnlock()

	pluginsDirChanged := s.PluginsDir != oldPluginsDir
	if pluginsDirChanged && a.pluginOperationInProgress() {
		return fmt.Errorf("cannot change plugins directory while a plugin install or removal is in progress")
	}

	a.mu.Lock()
	s.ClusterProxies = a.settings.ClusterProxies
	if err := config.Save(s); err != nil {
		a.mu.Unlock()
		return err
	}
	a.settings = s
	a.mu.Unlock()

	if pluginsDirChanged {
		a.onPluginsDirChanged()
		runtime.EventsEmit(a.ctx, "plugins:changed")
	}
	return nil
}

// SaveMarketplaceRepositories persists just the marketplace repository list,
// applied directly onto the authoritative in-memory settings under the lock.
// Unlike SaveSettings (which accepts a full settings snapshot from the
// frontend), this cannot be clobbered by, or clobber, a concurrent save of an
// unrelated settings section that was built from a stale cached snapshot.
// URLs are stored exactly as provided, no validation or canonicalization.
func (a *App) SaveMarketplaceRepositories(repos []config.MarketplaceRepository) error {
	for i := range repos {
		repos[i].URL = strings.TrimSpace(repos[i].URL)
	}

	a.mu.Lock()
	a.settings.MarketplaceRepositories = repos
	err := config.Save(a.settings)
	a.mu.Unlock()
	return err
}

// SavePluginsDir persists just the plugins directory, applied directly onto
// the authoritative in-memory settings under the lock (see
// SaveMarketplaceRepositories for why this avoids the SaveSettings
// full-object race). Rebuilds the plugin loader map if the directory changed.
func (a *App) SavePluginsDir(dir string) error {
	a.mu.RLock()
	oldDir := a.settings.PluginsDir
	a.mu.RUnlock()

	dirChanged := dir != oldDir
	if dirChanged && a.pluginOperationInProgress() {
		return fmt.Errorf("cannot change plugins directory while a plugin install or removal is in progress")
	}

	a.mu.Lock()
	a.settings.PluginsDir = dir
	err := config.Save(a.settings)
	a.mu.Unlock()
	if err != nil {
		return err
	}

	if dirChanged {
		a.onPluginsDirChanged()
		runtime.EventsEmit(a.ctx, "plugins:changed")
	}
	return nil
}

// GetClusterProxy returns the proxy settings for a specific cluster context.
func (a *App) GetClusterProxy(contextName string) config.ClusterProxy {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return a.settings.ClusterProxies[contextName]
}

// SaveClusterProxy persists proxy settings for a specific cluster context and
// evicts the cached clientset so the next Connect call rebuilds it with the new proxy.
func (a *App) SaveClusterProxy(contextName string, proxy config.ClusterProxy) error {
	a.mu.Lock()
	defer a.mu.Unlock()
	if a.settings.ClusterProxies == nil {
		a.settings.ClusterProxies = make(map[string]config.ClusterProxy)
	}
	a.settings.ClusterProxies[contextName] = proxy
	delete(a.clients, contextName)
	return config.Save(a.settings)
}

// GetSettings returns the persisted application settings. The first time the
// app runs (MarketplaceRepositories has never been persisted, i.e. still nil),
// the default marketplace source (config.GetMarketplaceBaseURL(), normally the
// built-in litelens repo, overridable via the MARKETPLACE_BASE_URL env var) is
// seeded as the first entry, so the frontend can show and let the user manage
// it like any other repository entry. Once any settings save persists a
// (possibly empty) MarketplaceRepositories array, that array is returned
// as-is on every subsequent call — if the user edits or removes that first
// entry, it is never re-injected.
func (a *App) GetSettings() (config.Settings, error) {
	s, err := config.Load()
	if err != nil {
		return config.Settings{}, err
	}

	if s.MarketplaceRepositories == nil {
		s.MarketplaceRepositories = []config.MarketplaceRepository{{URL: config.GetMarketplaceBaseURL()}}
	}
	return s, nil
}

// GetActiveKubeconfigPaths returns the kubeconfig files that client-go would load,
// respecting the KUBECONFIG env var with the same precedence rules the app uses to
// connect to clusters.
func (a *App) GetActiveKubeconfigPaths() ([]string, error) {
	rules := clientcmd.NewDefaultClientConfigLoadingRules()
	return rules.Precedence, nil
}

// IsPrivateRepoAccess reports whether the backend is configured for private-repo
// GitHub asset access (see config.IsPrivateRepoAccess), so the frontend can hide
// the Sandbox access-token UI when there is nothing for the user to configure.
func (a *App) IsPrivateRepoAccess() bool {
	return config.IsPrivateRepoAccess()
}

// GetContextKubeconfigPath returns the kubeconfig file path that defines the given context.
// Returns an empty string if the context is not found in any loaded kubeconfig.
func (a *App) GetContextKubeconfigPath(contextName string) (string, error) {
	a.mu.RLock()
	paths := a.settings.KubeconfigPaths
	a.mu.RUnlock()

	rules := kube.LoadingRules(paths)
	for _, path := range rules.Precedence {
		cfg, err := clientcmd.LoadFromFile(path)
		if err != nil {
			continue
		}
		if _, ok := cfg.Contexts[contextName]; ok {
			return path, nil
		}
	}
	return "", nil
}

// PickKubeconfigPath opens a native file dialog rooted at ~/.kube and returns the selected path.
func (a *App) PickKubeconfigPath() (string, error) {
	defaultDir := ""
	if home, err := os.UserHomeDir(); err == nil {
		kubeDir := filepath.Join(home, ".kube")
		if _, err := os.Stat(kubeDir); err == nil {
			defaultDir = kubeDir
		} else {
			defaultDir = home
		}
	}
	return runtime.OpenFileDialog(a.ctx, runtime.OpenDialogOptions{
		Title:            "Select Kubeconfig File",
		DefaultDirectory: defaultDir,
	})
}

// SaveLocaleTimezone persists the user's preferred timezone (IANA format, e.g. "America/New_York").
func (a *App) SaveLocaleTimezone(tz string) error {
	a.mu.Lock()
	a.settings.Locale = tz
	err := config.Save(a.settings)
	a.mu.Unlock()
	return err
}

// SaveKubeconfigPaths persists the list of kubeconfig paths and notifies the frontend.
func (a *App) SaveKubeconfigPaths(paths []string) error {
	a.mu.Lock()
	a.settings.KubeconfigPaths = paths
	err := config.Save(a.settings)
	a.mu.Unlock()
	if err != nil {
		return err
	}
	runtime.EventsEmit(a.ctx, "kubeconfig:changed")
	return nil
}

// PickPluginsDir opens a native folder dialog rooted at the current effective plugins directory.
func (a *App) PickPluginsDir() (string, error) {
	return runtime.OpenDirectoryDialog(a.ctx, runtime.OpenDialogOptions{
		Title:            "Select Plugins Directory",
		DefaultDirectory: a.pluginsRootDir(),
	})
}

package main

import (
	"embed"
	"log"
	goruntime "runtime"

	"github.com/joho/godotenv"
	"github.com/litelensapp/litelens/internal/app"
	"github.com/litelensapp/litelens/internal/config"
	"github.com/litelensapp/litelens/internal/plugin"
	"github.com/litelensapp/litelens/internal/storage"
	"github.com/litelensapp/litelens/internal/version"
	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/menu"
	"github.com/wailsapp/wails/v2/pkg/menu/keys"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"
	"github.com/wailsapp/wails/v2/pkg/options/mac"
)

//go:embed all:frontend/dist
var assets embed.FS

func main() {
	// Load .env if present — silently ignored when absent (e.g. production binary).
	// Existing environment variables are never overwritten.
	_ = godotenv.Load()

	// Enable dev mode storage if running a development build.
	storage.SetDevMode(Version == version.Dev)
	storage.SetRootDirOverride(config.GetRootDirOverride())

	a := app.NewApp(Version)

	// Build bind list: include only App, all plugins are accessed via InvokePlugin.
	bindList := []any{a}

	err := wails.Run(&options.App{
		Title:     "litelens",
		Width:     1024,
		Height:    768,
		MinWidth:  900,
		MinHeight: 600,
		AssetServer: &assetserver.Options{
			Assets:  assets,
			Handler: plugin.NewPluginAssetHandler(a.PluginAssetDir),
		},
		BackgroundColour: &options.RGBA{R: 27, G: 38, B: 54, A: 1},
		Menu:             buildMenu(a),
		Mac:              &mac.Options{},
		OnStartup:        a.Startup,
		OnDomReady:       a.DomReady,
		OnShutdown:       a.Shutdown,
		Bind:             bindList,
	})

	if err != nil {
		println("Error:", err.Error())
	}
}

func buildMenu(a *app.App) *menu.Menu {
	m := menu.NewMenu()

	// Build the app menu manually. On macOS the first submenu IS the app menu;
	// the OS replaces its title with the application name automatically.
	appSub := m.AddSubmenu("litelens")
	appSub.AddText("About LiteLens", nil, func(_ *menu.CallbackData) {
		go a.OpenAbout()
	})
	appSub.AddSeparator()
	appSub.AddText("Settings", keys.CmdOrCtrl("s"), func(_ *menu.CallbackData) {
		go a.OpenSettings()
	})
	appSub.AddSeparator()
	appSub.AddText("Quit", keys.CmdOrCtrl("q"), func(_ *menu.CallbackData) {
		go a.Quit()
	})

	// Edit menu — required for macOS to route clipboard shortcuts to the WKWebView.
	if goruntime.GOOS == "darwin" {
		// Native role: macOS binds Cut/Copy/Paste/etc. to the real cut:/copy:/paste:
		// selectors, which WKWebView handles as a trusted, first-party paste. Routing
		// Cmd+V through our own document.execCommand('paste') call instead (as the
		// manual items below do) makes WebKit treat it as an untrusted script-initiated
		// clipboard read and show its native "Paste" confirmation bubble before typing.
		m.Append(&menu.MenuItem{Role: menu.EditMenuRole})
	} else {
		editSub := m.AddSubmenu("Edit")
		editSub.AddText("Undo", keys.CmdOrCtrl("z"), func(_ *menu.CallbackData) {
			a.ExecJS("document.execCommand('undo')")
		})
		editSub.AddText("Redo", keys.Combo("z", keys.CmdOrCtrlKey, keys.ShiftKey), func(_ *menu.CallbackData) {
			a.ExecJS("document.execCommand('redo')")
		})
		editSub.AddSeparator()
		editSub.AddText("Cut", keys.CmdOrCtrl("x"), func(_ *menu.CallbackData) {
			a.ExecJS("document.execCommand('cut')")
		})
		editSub.AddText("Copy", keys.CmdOrCtrl("c"), func(_ *menu.CallbackData) {
			a.ExecJS("document.execCommand('copy')")
		})
		editSub.AddText("Paste", keys.CmdOrCtrl("v"), func(_ *menu.CallbackData) {
			a.ExecJS("document.execCommand('paste')")
		})
		editSub.AddSeparator()
		editSub.AddText("Select All", keys.CmdOrCtrl("a"), func(_ *menu.CallbackData) {
			a.ExecJS("document.execCommand('selectAll')")
		})
	}

	// View menu — window/webview controls.
	viewSub := m.AddSubmenu("View")
	viewSub.AddText("Reload", keys.CmdOrCtrl("r"), func(_ *menu.CallbackData) {
		a.ExecJS("location.reload()")
	})
	// Unlike Reload (webview-only), Full Reload restarts the whole Go process —
	// needed to recover state that only resets on a fresh process (e.g. the
	// one-shot startup update check).
	viewSub.AddText("Full Reload", keys.Combo("r", keys.CmdOrCtrlKey, keys.ShiftKey), func(_ *menu.CallbackData) {
		go func() {
			if err := a.RestartApp(); err != nil {
				log.Printf("main: restart app: %v", err)
			}
		}()
	})

	return m
}

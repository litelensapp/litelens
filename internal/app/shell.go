package app

import (
	"context"
	"log"
	"os"
	"os/exec"
	goruntime "runtime"
	"strings"
	"time"

	"github.com/litelensapp/litelens/internal/storage"
)

// GetDefaultShell returns the shell that will be used when ShellPath is not
// configured: $SHELL from the environment, falling back to /bin/zsh.
func (a *App) GetDefaultShell() string {
	if s := os.Getenv("SHELL"); s != "" {
		return s
	}
	return "/bin/zsh"
}

// GetAppDir returns the app's data directory (~/.litelens).
func (a *App) GetAppDir() string {
	return storage.Dir()
}

// OpenAppDir opens the app's data directory in the OS file manager
// (Finder on macOS, Explorer on Windows, xdg-open elsewhere).
func (a *App) OpenAppDir() error {
	dir := storage.Dir()
	var cmd *exec.Cmd
	switch goruntime.GOOS {
	case "darwin":
		cmd = exec.Command("open", dir)
	case "windows":
		cmd = exec.Command("explorer", dir)
	default:
		cmd = exec.Command("xdg-open", dir)
	}
	return cmd.Start()
}

// resolveLoginShellPATH queries the user's login shell for its full PATH and
// sets it on the current process so exec credential plugins (e.g. aws, gcloud)
// can be found. On non-macOS platforms this is a no-op. Failures are logged
// but never fatal — the app starts normally with the original PATH.
// shellPath overrides the auto-detected $SHELL when non-empty.
func resolveLoginShellPATH(shellPath string) {
	if goruntime.GOOS != "darwin" {
		return
	}
	shell := shellPath
	if shell == "" {
		shell = os.Getenv("SHELL")
	}
	if shell == "" {
		shell = "/bin/zsh"
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	out, err := exec.CommandContext(ctx, shell, "-l", "-c", "echo $PATH").Output()
	if err != nil {
		log.Printf("[env] login shell PATH resolution failed (shell: %q): %v", shell, err)
		return
	}
	if path := strings.TrimSpace(string(out)); path != "" {
		os.Setenv("PATH", path) //nolint:errcheck
		log.Printf("[env] PATH resolved from login shell %q", shell)
	}
}

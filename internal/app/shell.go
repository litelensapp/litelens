package app

import (
	"context"
	"log"
	"os"
	"os/exec"
	goruntime "runtime"
	"strings"
	"time"
)

// GetDefaultShell returns the shell that will be used when ShellPath is not
// configured: $SHELL from the environment, falling back to /bin/zsh.
func (a *App) GetDefaultShell() string {
	if s := os.Getenv("SHELL"); s != "" {
		return s
	}
	return "/bin/zsh"
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

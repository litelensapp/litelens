package app

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"context"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	goruntime "runtime"
	"strings"
	"time"

	"github.com/litelensapp/litelens/internal/config"
	"github.com/litelensapp/litelens/internal/lib/ratelimiter"
	"github.com/litelensapp/litelens/internal/plugin"
	"github.com/litelensapp/litelens/internal/updater"
	"github.com/wailsapp/wails/v2/pkg/runtime"
)

func (a *App) checkForUpdate() error {
	a.mu.RLock()
	token := a.settings.AccessToken
	a.mu.RUnlock()

	// Retry with bounded backoff: 3 attempts total, sleeping 5s then 10s between attempts
	var rel *updater.Release
	var err error
	sleeps := []time.Duration{5 * time.Second, 10 * time.Second}
	for attempt := range 3 {
		rel, err = updater.Check(a.version, token)
		if err == nil {
			// Success (either update available or no update needed)
			break
		}
		// Check if this is a rate-limit error; if so, don't retry
		if _, ok := errors.AsType[*ratelimiter.RateLimitError](err); ok {
			log.Printf("app: checkForUpdate: rate limited: %v", err)
			break
		}
		// Non-rate-limit failure; log and retry if we have attempts left
		log.Printf("app: checkForUpdate: attempt %d: %v", attempt+1, err)
		if attempt < len(sleeps) {
			time.Sleep(sleeps[attempt])
		}
	}

	if err != nil {
		// Either rate-limited or all retries exhausted
		if _, isRateLimitErr := err.(*ratelimiter.RateLimitError); isRateLimitErr {
			log.Printf("app: checkForUpdate: giving up due to rate limit")
		} else {
			log.Printf("app: checkForUpdate: giving up after 3 attempts")
		}
		return err
	}

	if rel == nil {
		// No update available (expected case when already up-to-date)
		runtime.EventsEmit(a.ctx, "update:check-complete", map[string]any{
			"updateAvailable": false,
		})
		return nil
	}

	// Update available; emit the event
	log.Printf("app: checkForUpdate: update available: %s", rel.TagName)
	runtime.EventsEmit(a.ctx, "update:available", map[string]any{
		"latestVersion": rel.TagName,
		"releaseURL":    rel.HTMLURL,
		"releaseNotes":  rel.Body,
		"assetURL":      rel.AssetURL,
		"downloadSize":  config.FormatBytes(rel.DownloadSize),
	})
	return nil
}

// CheckForUpdate manually triggers a check for app updates. It runs synchronously
// and emits the update:available event if a new version is found. Returns an error
// if the check fails after all retries.
func (a *App) CheckForUpdate() error {
	return a.checkForUpdate()
}

// selectUpdateStrategy returns the update strategy name for a given OS.
// This function is extracted for testability (to verify OS branching without
// executing platform-specific code).
func selectUpdateStrategy(goos string) string {
	switch goos {
	case "windows":
		return "windows"
	case "linux":
		return "linux"
	case "darwin":
		return "macos"
	default:
		return "unsupported"
	}
}

// PerformUpdate downloads and installs the new version, then relaunches the
// app and quits the current instance. Windows has no bash/install-script
// support, so it downloads the release asset directly and swaps the binary
// via a detached helper script; macOS runs the bash install script (which
// handles code-signing); Linux downloads the binary and uses pkexec to run
// the install-helper with elevated privileges (avoiding TTY issues with sudo).
func (a *App) PerformUpdate(version string) error {
	a.mu.RLock()
	token := a.settings.AccessToken
	a.mu.RUnlock()

	switch selectUpdateStrategy(goruntime.GOOS) {
	case "windows":
		if err := a.performWindowsUpdate(version, token); err != nil {
			return err
		}
		runtime.Quit(a.ctx)
		return nil
	case "linux":
		if err := a.performLinuxUpdate(version, token); err != nil {
			return err
		}
		// The install-helper always installs to this hardcoded path. Do NOT use
		// os.Executable() here: by this point the helper has renamed the running
		// process's original binary to a .backup file and then deleted it, so
		// os.Executable() (which resolves /proc/self/exe) returns a path that no
		// longer exists, causing the relaunch to silently fail.
		if err := exec.Command("/usr/local/bin/litelens").Start(); err != nil {
			fmt.Fprintf(os.Stderr, "ERROR: failed to relaunch updated app: %v\n", err)
		}
		runtime.Quit(a.ctx)
		return nil
	}

	// macOS path (darwin)
	scriptURL := config.GetInstallScriptURL()

	req, err := http.NewRequestWithContext(a.ctx, http.MethodGet, scriptURL, nil)
	if err != nil {
		return fmt.Errorf("download install script: %w", err)
	}
	req.Header.Set("Accept", "application/vnd.github.raw+json")
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("download install script: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("download install script: HTTP %d", resp.StatusCode)
	}

	f, err := os.CreateTemp("", "litelens-install-*.sh")
	if err != nil {
		return fmt.Errorf("create temp file: %w", err)
	}
	defer os.Remove(f.Name())

	if _, err = io.Copy(f, resp.Body); err != nil {
		f.Close()
		return fmt.Errorf("write install script: %w", err)
	}
	f.Close()

	// macOS .app bundles inherit a stripped PATH. Build an explicit one so the
	// install script can find all standard tools (curl, unzip, codesign, etc.)
	// without relying on whatever the bundle's environment happens to include.
	env := make([]string, 0, len(os.Environ())+1)
	for _, e := range os.Environ() {
		if !strings.HasPrefix(e, "PATH=") {
			env = append(env, e)
		}
	}
	env = append(env, "PATH=/opt/homebrew/bin:/usr/local/bin:/usr/bin:/bin:/usr/sbin:/sbin")
	if token != "" {
		env = append(env, "LITELENS_ACCESS_TOKEN="+token)
	}

	// Capture stderr so we can surface the actual error message to the user
	// instead of the opaque "exit status N" that cmd.Run returns.
	var stderrBuf bytes.Buffer
	cmd := exec.Command("/bin/bash", f.Name(), version)
	cmd.Stdout = os.Stdout
	cmd.Stderr = io.MultiWriter(os.Stderr, &stderrBuf)
	cmd.Env = env
	if err = cmd.Run(); err != nil {
		if msg := strings.TrimSpace(stderrBuf.String()); msg != "" {
			return fmt.Errorf("install failed: %s", msg)
		}
		return fmt.Errorf("install failed: %w", err)
	}

	exec.Command("open", "-n", "/Applications/LiteLens.app").Start() //nolint:errcheck
	runtime.Quit(a.ctx)
	return nil
}

// performLinuxUpdate downloads the Linux release binary and uses pkexec to
// invoke the litelens-install-helper to swap it into place. pkexec handles
// authentication via the polkit agent (system dialog), avoiding TTY issues
// that would occur if we tried to use sudo from a non-interactive context.
func (a *App) performLinuxUpdate(version, token string) error {
	rel, err := updater.FetchRelease(version, token)
	if err != nil {
		return fmt.Errorf("resolve release: %w", err)
	}
	if rel.AssetURL == "" {
		return fmt.Errorf("no linux asset found for release %s", version)
	}

	// Check if a SHA256 checksum is available in the release assets
	// (similar to the plugin release pattern). If no checksum is available,
	// fail the update with a clear error to maintain security posture.
	// Pass the already-fetched release object to avoid double-fetching.
	checksumHex, err := findReleaseChecksumInRelease(a.ctx, rel, token)
	if err != nil {
		return fmt.Errorf("checksum verification unavailable: %w", err)
	}
	if checksumHex == "" {
		return fmt.Errorf("update failed: release has no integrity verification (missing SHA256 checksum asset)")
	}

	req, err := http.NewRequestWithContext(a.ctx, http.MethodGet, rel.AssetURL, nil)
	if err != nil {
		return fmt.Errorf("download binary: %w", err)
	}
	req.Header.Set("Accept", "application/octet-stream")
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("download binary: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("download binary: HTTP %d", resp.StatusCode)
	}

	tmpFile, err := os.CreateTemp("", "litelens-binary-*.tmp")
	if err != nil {
		return fmt.Errorf("create temp file: %w", err)
	}
	defer os.Remove(tmpFile.Name())

	if _, err = io.Copy(tmpFile, resp.Body); err != nil {
		tmpFile.Close()
		return fmt.Errorf("write binary: %w", err)
	}
	tmpFile.Close()

	// Verify the binary integrity against the fetched checksum before
	// performing any privileged operations.
	if err := plugin.VerifySHA256(tmpFile.Name(), checksumHex); err != nil {
		return fmt.Errorf("binary verification failed: %w", err)
	}

	// The downloaded release asset is a tar.gz archive bundling litelens,
	// litelens-install-helper, and appicon.png; extract just the litelens
	// binary before handing it to the install-helper.
	extractedPath, err := extractBinaryFromTarGz(tmpFile.Name())
	if err != nil {
		return fmt.Errorf("extract binary from archive: %w", err)
	}
	defer os.Remove(extractedPath)

	// Make the extracted binary readable (not executable yet; install-helper will chmod it).
	if err := os.Chmod(extractedPath, 0o644); err != nil {
		return fmt.Errorf("chmod extracted binary: %w", err)
	}

	// Resolve the install-helper to an absolute path to prevent PATH hijacking.
	// The helper is shipped in the same directory as the running litelens binary.
	helperPath, err := resolveInstallerHelper()
	if err != nil {
		return fmt.Errorf("install helper not found: %w", err)
	}

	// Invoke litelens-install-helper via pkexec to perform the installation.
	var stderrBuf bytes.Buffer
	cmd := exec.Command("pkexec", helperPath,
		"--binary", extractedPath,
		"--version", version)
	cmd.Stdout = os.Stdout
	cmd.Stderr = io.MultiWriter(os.Stderr, &stderrBuf)

	if err = cmd.Run(); err != nil {
		// Distinguish pkexec authentication cancellation (exit code 126)
		// from helper-not-found (exit code 127, though this is now caught earlier).
		if exitErr, ok := err.(*exec.ExitError); ok {
			code := exitErr.ExitCode()
			switch code {
			case 126:
				return fmt.Errorf("update cancelled: authentication was not granted")
			case 127:
				return fmt.Errorf("update failed: install helper not found or not executable")
			}
		}

		if msg := strings.TrimSpace(stderrBuf.String()); msg != "" {
			return fmt.Errorf("install failed: %s", msg)
		}
		return fmt.Errorf("install failed: %w", err)
	}

	return nil
}

// extractBinaryFromTarGz opens a gzip-compressed tar archive and extracts
// only the "litelens" entry to a new temp file. Other entries in the
// archive (e.g. litelens-install-helper, appicon.png) are skipped. Returns
// the path to the extracted temp file; the caller must os.Remove() it.
func extractBinaryFromTarGz(tarGzPath string) (extractedPath string, err error) {
	f, err := os.Open(tarGzPath)
	if err != nil {
		return "", fmt.Errorf("opening archive: %w", err)
	}
	defer f.Close()

	gzr, err := gzip.NewReader(f)
	if err != nil {
		return "", fmt.Errorf("decompressing gzip: %w", err)
	}
	defer gzr.Close()

	tr := tar.NewReader(gzr)
	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			return "", fmt.Errorf("litelens entry not found in archive")
		}
		if err != nil {
			return "", fmt.Errorf("reading tar: %w", err)
		}

		if hdr.Name != "litelens" {
			continue
		}

		if hdr.Typeflag != tar.TypeReg {
			return "", fmt.Errorf("litelens entry is not a regular file (type %v)", hdr.Typeflag)
		}

		tmp, err := os.CreateTemp("", "litelens-extracted-*.tmp")
		if err != nil {
			return "", fmt.Errorf("create temp file for extraction: %w", err)
		}
		defer tmp.Close()

		if _, err := io.Copy(tmp, tr); err != nil {
			os.Remove(tmp.Name())
			return "", fmt.Errorf("extract binary: %w", err)
		}

		return tmp.Name(), nil
	}
}

// findReleaseChecksumInRelease searches a pre-fetched release object for a SHA256
// checksum file matching the current platform's release asset (as resolved by
// updater.FetchRelease/rel.AssetURL). Returns the checksum hex string.
// If no checksum is found, returns ("", nil) — the caller must decide
// whether to fail or proceed without verification.
func findReleaseChecksumInRelease(ctx context.Context, rel *updater.Release, token string) (string, error) {
	// When Assets is empty, the release was resolved via the unauthenticated
	// public path (no token configured), which never calls the GitHub API
	// and so never populates Assets. The checksum asset still exists at the
	// well-known convention scripts/install.sh uses: the download asset's
	// name with a ".sha256" suffix.
	if len(rel.Assets) == 0 {
		return downloadChecksum(ctx, rel.AssetURL+".sha256", token)
	}

	// The checksum file name mirrors the platform asset name with a ".sha256"
	// suffix, e.g. "litelens-windows-amd64.exe.sha256".
	checksumName := ""
	for _, asset := range rel.Assets {
		assetURL := asset.URL
		if !config.IsPrivateRepoAccess() {
			assetURL = asset.BrowserDownloadURL
		}
		if assetURL != "" && assetURL == rel.AssetURL {
			checksumName = asset.Name + ".sha256"
			break
		}
	}

	if checksumName == "" {
		// No checksum available; this is security-critical.
		return "", nil
	}

	// Find the checksum asset URL.
	checksumAssetURL := ""
	for _, asset := range rel.Assets {
		if asset.Name == checksumName {
			if config.IsPrivateRepoAccess() {
				checksumAssetURL = asset.URL
			} else {
				checksumAssetURL = asset.BrowserDownloadURL
			}
			break
		}
	}

	if checksumAssetURL == "" {
		// Checksum file was expected but not found in assets.
		return "", nil
	}

	return downloadChecksum(ctx, checksumAssetURL, token)
}

// downloadChecksum fetches a SHA256 checksum file (tiny, just a hex string +
// newline) from the given URL and returns its trimmed contents.
func downloadChecksum(ctx context.Context, checksumAssetURL, token string) (string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, checksumAssetURL, nil)
	if err != nil {
		return "", fmt.Errorf("create checksum request: %w", err)
	}
	req.Header.Set("Accept", "application/octet-stream")
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("download checksum: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("checksum download returned HTTP %d", resp.StatusCode)
	}

	checksumBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("read checksum: %w", err)
	}

	return strings.TrimSpace(string(checksumBytes)), nil
}

// resolveInstallerHelper locates the litelens-install-helper binary.
// It first checks /usr/local/bin (the trusted, production install location),
// then falls back to the directory of the currently-running executable (for dev/testing).
// Returns an absolute path or an error if not found.
// Prioritizing /usr/local/bin prevents execution of a helper from a user-writable
// directory (e.g., /tmp/build or a user's home), which would be a privilege escalation risk.
func resolveInstallerHelper() (string, error) {
	// Check the standard installed location first (root-owned, trusted location).
	helperPath := "/usr/local/bin/litelens-install-helper"
	if info, err := os.Stat(helperPath); err == nil && info.Mode().IsRegular() {
		return helperPath, nil
	}

	// Fall back to the directory of the currently-running executable (dev/testing only).
	exePath, err := os.Executable()
	if err == nil {
		// Same directory as the running litelens binary.
		exeDir := filepath.Dir(exePath)
		exeDirHelperPath := filepath.Join(exeDir, "litelens-install-helper")
		if info, err := os.Stat(exeDirHelperPath); err == nil && info.Mode().IsRegular() {
			return exeDirHelperPath, nil
		}
	}

	return "", fmt.Errorf("litelens-install-helper not found in /usr/local/bin or executable directory")
}

// copyFile copies src to dst, creating dst if it doesn't exist. Used instead
// of os.Rename because src and dst may be on different volumes/temp dirs.
func copyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()

	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	if _, err := io.Copy(out, in); err != nil {
		out.Close()
		return err
	}
	return out.Close()
}

// performWindowsUpdate downloads the Windows release asset for version and
// swaps it in for the currently-running executable. A running .exe can't be
// overwritten or renamed out from under itself on Windows, so a detached
// helper batch script waits for this process to exit, moves the new binary
// into place, relaunches it, then deletes itself.
func (a *App) performWindowsUpdate(version, token string) error {
	rel, err := updater.FetchRelease(version, token)
	if err != nil {
		return fmt.Errorf("resolve release: %w", err)
	}
	if rel.AssetURL == "" {
		return fmt.Errorf("no windows asset found for release %s", version)
	}

	// Fail closed if no checksum is published for this release asset, matching
	// the integrity guarantee already enforced on the Linux update path.
	checksumHex, err := findReleaseChecksumInRelease(a.ctx, rel, token)
	if err != nil {
		return fmt.Errorf("checksum verification unavailable: %w", err)
	}
	if checksumHex == "" {
		return fmt.Errorf("update failed: release has no integrity verification (missing SHA256 checksum asset)")
	}

	req, err := http.NewRequestWithContext(a.ctx, http.MethodGet, rel.AssetURL, nil)
	if err != nil {
		return fmt.Errorf("download update: %w", err)
	}
	req.Header.Set("Accept", "application/octet-stream")
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("download update: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("download update: HTTP %d", resp.StatusCode)
	}

	exePath, err := os.Executable()
	if err != nil {
		return fmt.Errorf("resolve current executable: %w", err)
	}
	// The path is interpolated into a generated .bat file below; reject
	// anything containing batch metacharacters rather than trying to quote
	// around them.
	const unsafeChars = "&|<>^%!\""
	if strings.ContainsAny(exePath, unsafeChars) {
		return fmt.Errorf("executable path contains unsupported characters: %s", exePath)
	}

	// Download to a temp file first so it can be checksum-verified before
	// being swapped in as the running executable's replacement.
	tmpFile, err := os.CreateTemp("", "litelens-update-*.exe")
	if err != nil {
		return fmt.Errorf("create temp file: %w", err)
	}
	defer os.Remove(tmpFile.Name())

	if _, err = io.Copy(tmpFile, resp.Body); err != nil {
		tmpFile.Close()
		return fmt.Errorf("write update: %w", err)
	}
	tmpFile.Close()

	if err := plugin.VerifySHA256(tmpFile.Name(), checksumHex); err != nil {
		return fmt.Errorf("binary verification failed: %w", err)
	}

	newExe := exePath + ".new"
	if err := copyFile(tmpFile.Name(), newExe); err != nil {
		os.Remove(newExe)
		return fmt.Errorf("write update: %w", err)
	}

	// Wait for this process's PID to disappear from tasklist (bounded to
	// ~30s so a misdetection can't hang the update forever), then swap the
	// binary in and relaunch. The script tries to delete itself last; that
	// delete commonly fails silently (Windows won't let a running .bat
	// remove itself), leaving a harmless leftover file in TempDir.
	script := fmt.Sprintf(
		"@echo off\r\n"+
			"setlocal enabledelayedexpansion\r\n"+
			"set count=0\r\n"+
			":wait\r\n"+
			"tasklist /FI \"PID eq %d\" 2>NUL | find \"%d\" >NUL\r\n"+
			"if not errorlevel 1 (\r\n"+
			"  set /a count+=1\r\n"+
			"  if !count! geq 30 goto done\r\n"+
			"  timeout /t 1 /nobreak >NUL\r\n"+
			"  goto wait\r\n"+
			")\r\n"+
			":done\r\n"+
			"move /y \"%s\" \"%s\"\r\n"+
			"start \"\" \"%s\"\r\n"+
			"del \"%%~f0\"\r\n",
		os.Getpid(), os.Getpid(), newExe, exePath, exePath,
	)
	scriptPath := filepath.Join(os.TempDir(), "litelens-update.bat")
	if err = os.WriteFile(scriptPath, []byte(script), 0o700); err != nil {
		os.Remove(newExe)
		return fmt.Errorf("write update script: %w", err)
	}

	cmd := exec.Command("cmd", "/C", "start", "/min", "", scriptPath)
	if err = cmd.Start(); err != nil {
		os.Remove(newExe)
		os.Remove(scriptPath)
		return fmt.Errorf("launch update script: %w", err)
	}

	return nil
}

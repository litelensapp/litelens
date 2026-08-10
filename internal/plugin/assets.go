package plugin

import (
	"net/http"
	"path/filepath"
	"strings"
)

// NewPluginAssetHandler creates an HTTP handler for serving plugin assets.
// Routes /api/plugins/{pluginID}/* to {resolvePluginDir(pluginID)}/* with
// path-traversal protection. Callers include the "dist/" segment themselves
// (e.g. /api/plugins/helm/dist/index.js), so it must not be added again here
// — doing so previously produced a nonexistent .../dist/dist/index.js path
// and made every plugin bundle 404. resolvePluginDir is called per-request
// (not captured once) and must return the plugin's actual on-disk directory
// rather than assuming {pluginsRootDir}/{pluginID} — the on-disk directory
// name is allowed to differ from the plugin ID (e.g. a local dev build under
// plugins/helm/.output/ with id "helm"; see App.PluginAssetDir).
func NewPluginAssetHandler(resolvePluginDir func(pluginID string) (string, bool)) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Only handle GET requests for plugin assets
		if r.Method != http.MethodGet {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		path := r.URL.Path
		if !strings.HasPrefix(path, "/api/plugins/") {
			http.Error(w, "Not found", http.StatusNotFound)
			return
		}

		// Parse /api/plugins/{pluginID}/path/to/file
		parts := strings.Split(path, "/")
		if len(parts) < 4 || parts[3] == "" {
			http.Error(w, "Invalid plugin path", http.StatusBadRequest)
			return
		}

		pluginID := parts[3]
		relativePath := strings.Join(parts[4:], "/")

		// Build the full disk path
		pluginDistDir, ok := resolvePluginDir(pluginID)
		if !ok {
			http.Error(w, "Not found", http.StatusNotFound)
			return
		}
		fullPath := filepath.Join(pluginDistDir, relativePath)

		// Validate path traversal — ensure resolved path is within dist dir
		// Use filepath.Abs and filepath.Rel to safely check containment
		absPluginDistDir, err := filepath.Abs(pluginDistDir)
		if err != nil {
			http.Error(w, "Internal server error", http.StatusInternalServerError)
			return
		}

		absFullPath, err := filepath.Abs(fullPath)
		if err != nil {
			http.Error(w, "Internal server error", http.StatusInternalServerError)
			return
		}

		// Check if resolved path is within the dist directory
		relPath, err := filepath.Rel(absPluginDistDir, absFullPath)
		if err != nil || strings.HasPrefix(relPath, "..") {
			http.Error(w, "Forbidden", http.StatusForbidden)
			return
		}

		// Serve the file with appropriate content type
		// Set Content-Type to application/javascript for JS files, otherwise infer
		if strings.HasSuffix(fullPath, ".js") {
			w.Header().Set("Content-Type", "application/javascript; charset=utf-8")
		}

		// Use http.ServeFile which handles range requests, etags, etc.
		http.ServeFile(w, r, fullPath)
	})
}

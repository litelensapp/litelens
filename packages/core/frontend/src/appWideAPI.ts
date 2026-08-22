import type { QueryClient } from "@tanstack/react-query";

/**
 * Register the plugin's global stylesheet(s) with the host. Unlike
 * clusterWideAPI.registerViews, this isn't scoped per-view — a plugin's
 * compiled CSS (e.g. Tailwind output) applies across all of its views, so
 * it's registered once per plugin rather than attached to each view config.
 *
 * Example:
 *   appWideAPI.registerStylesheets("helm", [import("./style.css")]);
 *
 * This function is replaced at runtime by the host's actual implementation
 * (injected in frontend/src/expose/index.tsx). If you see this error,
 * registerStylesheets was called before the host initialized the injection.
 */
function registerStylesheets(
  // eslint-disable-next-line @typescript-eslint/no-unused-vars
  pluginId: string,
  // eslint-disable-next-line @typescript-eslint/no-unused-vars
  stylesheets: Array<Promise<{ default: string }>>
): void {
  // This function is replaced at runtime by the host's actual implementation.
  // If you see this error, registerStylesheets was called before the host injected the real implementation.
  throw new Error(
    "appWideAPI.registerStylesheets is not available. This function must be imported from " +
      "'@litelens/core' within a plugin bundle loaded by the litelens host."
  );
}

/**
 * Get the host's singleton QueryClient instance. Useful for plugin code
 * that isn't a mounted React component (e.g. module-scope registration in
 * index.ts) and therefore can't call useQueryClient()'s hook.
 *
 * Example:
 *   const queryClient = appWideAPI.getQueryClient();
 *   queryClient.invalidateQueries({ queryKey: [...] });
 *
 * This function is replaced at runtime by the host's actual implementation
 * (injected in frontend/src/expose/index.tsx). If you see this error,
 * getQueryClient was called before the host initialized the injection.
 */
function getQueryClient(): QueryClient {
  throw new Error(
    "appWideAPI.getQueryClient is not available. This function must be imported from " +
      "'@litelens/core' within a plugin bundle loaded by the litelens host."
  );
}

/**
 * App-wide capabilities exposed to plugins — not scoped to the single-cluster
 * view, unlike clusterWideAPI (see useExposeProperties() for that constraint).
 */
export const appWideAPI = {
  registerStylesheets,
  getQueryClient,
};

/**
 * @litelens/core — React hooks for litelens plugin development
 *
 * This package exports hooks that plugin frontends can consume. The implementations
 * are provided by the host at runtime via import injection.
 */

/**
 * Hook to get callbacks for opening resource detail drawers by kind.
 * Returns a mapping of resource kinds (lowercase) to handler functions.
 *
 * Example:
 *   const links = useResourceLinks();
 *   links.pod(namespace, podName);  // Opens pod detail drawer
 *
 * This signature is duplicated from the host's real implementation at
 * frontend/src/app/clusters/shared/hooks/useResourceLinks.ts (which this
 * package can't import directly — see main.tsx's vendor injection). If that
 * hook's return type changes, update this signature to match.
 */
export function useResourceLinks(): Record<string, (namespace: string, name: string) => void> {
  // This function is replaced at runtime by the host's actual implementation.
  // If you see this error, useResourceLinks was called outside of a plugin context.
  throw new Error(
    "useResourceLinks is not available. This hook must be imported from '@litelens/core' " +
      "within a plugin bundle loaded by the litelens host."
  );
}

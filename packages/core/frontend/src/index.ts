import type { ComponentType } from "react";
import type { NavEntry, SharedUnifiedTrayContentProps } from "./types";

export * from "./types";

/**
 * @litelens/core — React hooks for litelens plugin development
 *
 * This package exports hooks that plugin frontends can consume. The implementations
 * are provided by the host at runtime via import injection.
 */

/**
 * Hook to access cluster-scoped APIs exposed to plugins. Returns a collection
 * of capabilities valid only within the single-cluster view — i.e. inside
 * MainLayout's subtree (MainLayoutProvider → DetailDrawerProvider). Calling
 * this from a component rendered outside that subtree (e.g. Settings,
 * Marketplace, or any future app-wide screen) will throw, because the
 * underlying host state (DetailDrawerContext) has no provider there. A future
 * useAppWideAPI() will cover capabilities that don't have this constraint.
 *
 * Example:
 *   const { activeContext, activeNamespaces, activeResource, availableNamespaces, onNavigateToView, resourceLinks, unifiedTray, useRegisterClusterWideEvents, useRegisterNavEntry, useRegisterTrayFamilies } = useClusterWideAPI();
 *   resourceLinks.pod(namespace, podName);  // Opens pod detail drawer
 *   unifiedTray.openTab("my-plugin-family", { label: "...", dedupeKey: "..." });
 *   useRegisterClusterWideEvents({
 *     "helm:release-updated": (payload) => { console.log("Release updated:", payload); },
 *   });
 *   useRegisterNavEntry("helm", "Helm", HELM_NAV_ENTRY);
 *   useRegisterTrayFamilies("helm", HELM_TRAY_FAMILIES);
 *
 * This signature is duplicated from the host's real implementation at
 * frontend/src/expose/hooks/useClusterWideAPI.ts (which this package can't
 * import directly — see main.tsx's vendor injection). If that hook's return
 * type changes, update this signature to match.
 */
export function useClusterWideAPI(): {
  activeContext: string;
  activeNamespaces: string[];
  activeResource: string;
  availableNamespaces: Array<{ Name: string }>;
  onNavigateToView: (view: string) => void;
  resourceLinks: Record<string, (namespace: string, name: string) => void>;
  unifiedTray: { openTab: (family: string, params: unknown) => void } | null;
  useRegisterClusterWideEvents: (handlers: Record<string, (payload: unknown) => void>) => void;
  useRegisterNavEntry: (
    pluginId: string,
    pluginName: string,
    navEntry: NavEntry<string> | undefined
  ) => void;
  useRegisterTrayFamilies: (
    pluginId: string,
    families: Record<string, ComponentType<SharedUnifiedTrayContentProps>> | undefined
  ) => void;
} {
  // This function is replaced at runtime by the host's actual implementation.
  // If you see this error, useClusterWideAPI was called outside of a plugin context.
  throw new Error(
    "useClusterWideAPI is not available. This hook must be imported from '@litelens/core' " +
      "within a plugin bundle loaded by the litelens host."
  );
}

import { useEffect } from "react";
import type { ComponentType } from "react";
import type { SharedUnifiedTrayContentProps } from "@litelens/core";
import { registerTrayFamilies, unregisterTrayFamilies } from "./pluginTrayRegistry";

/**
 * Exposed to plugins via useClusterWideAPI(). A plugin calls this from its own
 * PluginView (the same component already dynamically imported for rendering
 * — see PluginResourceView, which keeps every non-uninstalled plugin's view
 * mounted, hidden when inactive) to push its own tray-family content
 * components into the host's unified tray. The host never imports a
 * plugin-specific tray contract — it only ever touches the existing
 * PLUGIN_VIEW/PLUGIN_STYLES exports.
 */
export function useRegisterTrayFamilies(
  pluginId: string,
  families: Record<string, ComponentType<SharedUnifiedTrayContentProps>> | undefined
): void {
  useEffect(() => {
    if (!families) return;
    registerTrayFamilies(pluginId, families);
    return () => unregisterTrayFamilies(pluginId);
  }, [pluginId, families]);
}

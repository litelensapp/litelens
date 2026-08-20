import { useEffect } from "react";
import type { NavEntry } from "@litelens/core";
import { registerNavEntry, unregisterNavEntry } from "./pluginNavRegistry";

/**
 * Exposed to plugins via useClusterWideAPI(). A plugin calls this from its own
 * PluginView (the same component already dynamically imported for rendering
 * — see PluginResourceView, which keeps every non-uninstalled plugin's view
 * mounted, hidden when inactive) to push its own nav entry into the host's
 * sidebar. The host never imports a plugin-specific nav contract — it only
 * ever touches the existing PLUGIN_VIEW/PLUGIN_STYLES exports.
 */
export function useRegisterNavEntry(
  pluginId: string,
  pluginName: string,
  navEntry: NavEntry<string> | undefined
): void {
  useEffect(() => {
    if (!navEntry) return;
    registerNavEntry(pluginId, pluginName, navEntry);
    return () => unregisterNavEntry(pluginId);
  }, [pluginId, pluginName, navEntry]);
}

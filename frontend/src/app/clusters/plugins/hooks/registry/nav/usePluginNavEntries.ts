import type { NavEntry } from "@litelens/core";
import { useMemo, useSyncExternalStore } from "react";
import { getNavEntries, subscribeNavRegistry } from "./pluginNavRegistry";

interface PluginNavData {
  navEntries: NavEntry<string>[];
  viewTypeToPluginId: Record<string, string>;
  pluginNameByViewType: Record<string, string>;
  resourceLabels: Record<string, string>;
}

/**
 * Reactive read side of pluginNavRegistry — rebuilds the merged nav data
 * (mirroring the shape useGetInstalledPluginNav used to derive from dynamic
 * imports) whenever a plugin registers/unregisters via useRegisterNavEntry.
 */
export function usePluginNavEntries(): PluginNavData {
  const registered = useSyncExternalStore(subscribeNavRegistry, getNavEntries, getNavEntries);

  return useMemo<PluginNavData>(() => {
    const navEntries: NavEntry<string>[] = [];
    const viewTypeToPluginId: Record<string, string> = {};
    const pluginNameByViewType: Record<string, string> = {};
    const resourceLabels: Record<string, string> = {};

    for (const { pluginId, pluginName, navEntry } of registered) {
      navEntries.push(navEntry);

      if (navEntry.kind === "group") {
        for (const item of navEntry.group.items) {
          if (item.view) {
            viewTypeToPluginId[item.view] = pluginId;
            pluginNameByViewType[item.view] = pluginName;
            resourceLabels[item.view] = item.label;
          }
        }
      } else if (navEntry.kind === "item" && navEntry.item.view) {
        const view = navEntry.item.view;
        viewTypeToPluginId[view] = pluginId;
        pluginNameByViewType[view] = pluginName;
        resourceLabels[view] = navEntry.item.label;
      }
    }

    return { navEntries, viewTypeToPluginId, pluginNameByViewType, resourceLabels };
  }, [registered]);
}

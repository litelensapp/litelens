import { useQuery } from "@tanstack/react-query";
import { NavEntry } from "@litelens/design-system";
import { useMemo } from "react";
import { useGetInstalledPlugins } from "../../../marketplace/hooks/useGetInstalledPlugins";
import { ensurePluginStylesheet } from "../utils/ensurePluginStylesheet";
import { loadPluginModule } from "../utils/loadPluginModule";

interface PluginNavData {
  navEntries: NavEntry<string>[];
  viewTypeToPluginId: Record<string, string>;
  pluginNameByViewType: Record<string, string>;
  resourceLabels: Record<string, string>;
}

interface UseGetInstalledPluginNavResult {
  pluginNavData: PluginNavData;
  isLoading: boolean;
}

/**
 * Discovers installed plugins at runtime and dynamically imports their nav entries.
 * Queries the backend for the full status of every installed plugin in one call
 * via useGetInstalledPlugins, then for each READY plugin:
 * 1. Dynamically imports its bundle (with cache-busted URL using the returned bundleChecksum)
 * 2. Reads PLUGIN_NAV_ENTRY export
 * 3. Builds a merged viewType→pluginId mapping and list of all nav entries
 *
 * This ensures the app never knows about plugin-specific nav structure at build time;
 * it's all discovered at runtime from the plugin's own export. Plugin discovery is
 * independent of marketplace availability — already-installed plugins populate nav
 * even if the marketplace is unreachable/offline.
 */
export const useGetInstalledPluginNav = (): UseGetInstalledPluginNavResult => {
  const { readyPlugins, isLoading: isLoadingStatuses } = useGetInstalledPlugins();

  // Fetch nav entries from each READY plugin's bundle via dynamic import
  const { data: navEntriesData = [] } = useQuery({
    queryKey: [
      "plugin-nav-entries",
      readyPlugins.map((s) => `${s.pluginId}:${s.bundleChecksum}`).join(","),
    ],
    queryFn: async () => {
      return Promise.all(
        readyPlugins.map(async (status) => {
          try {
            const module = await loadPluginModule(status.pluginId, status.bundleChecksum);
            ensurePluginStylesheet(status.pluginId, module.PLUGIN_STYLES);
            const navEntry = module.PLUGIN_NAV_ENTRY;
            return {
              pluginId: status.pluginId,
              pluginName: status.pluginId,
              navEntry,
            };
          } catch (error) {
            console.error(`Failed to load nav entry for plugin ${status.pluginId}:`, error);
            return null;
          }
        })
      );
    },
    enabled: readyPlugins.length > 0,
  });

  // Build the merged nav data from all plugin entries
  const pluginNavData = useMemo<PluginNavData>(() => {
    const navEntries: NavEntry<string>[] = [];
    const viewTypeToPluginId: Record<string, string> = {};
    const pluginNameByViewType: Record<string, string> = {};
    const resourceLabels: Record<string, string> = {};

    for (const data of navEntriesData) {
      if (!data?.navEntry) continue;

      navEntries.push(data.navEntry);

      // Walk the nav entry to build viewType→pluginId and resource labels
      if (data.navEntry.kind === "group") {
        const { items } = data.navEntry.group;
        for (const item of items) {
          if (item.view) {
            viewTypeToPluginId[item.view] = data.pluginId;
            pluginNameByViewType[item.view] = data.pluginName;
            resourceLabels[item.view] = item.label;
          }
        }
      } else if (data.navEntry.kind === "item") {
        if (data.navEntry.item.view) {
          viewTypeToPluginId[data.navEntry.item.view] = data.pluginId;
          pluginNameByViewType[data.navEntry.item.view] = data.pluginName;
          resourceLabels[data.navEntry.item.view] = data.navEntry.item.label;
        }
      }
    }

    return {
      navEntries,
      viewTypeToPluginId,
      pluginNameByViewType,
      resourceLabels,
    };
  }, [navEntriesData]);

  const isLoading = isLoadingStatuses;

  return {
    pluginNavData,
    isLoading,
  };
};

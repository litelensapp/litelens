import { useQuery } from "@tanstack/react-query";
import type { UnifiedTrayContentComponent } from "../../shared/components/trays/unified/UnifiedTrayTypes";
import { useGetInstalledPlugins } from "../../../marketplace/hooks/useGetInstalledPlugins";
import { loadPluginModule } from "../utils/loadPluginModule";

/**
 * Discovers each installed (READY) plugin's own tray-family content
 * components at runtime by reading the `PLUGIN_TRAY_FAMILIES` export from
 * its dynamically-imported bundle. The host never has static knowledge of
 * plugin-owned tray family names or their param shapes.
 */
export const usePluginTrayRegistry = (): Record<string, UnifiedTrayContentComponent> => {
  const { readyPlugins } = useGetInstalledPlugins();

  const { data: trayFamilies = [] } = useQuery({
    queryKey: [
      "plugin-tray-families",
      readyPlugins.map((s) => `${s.pluginId}:${s.bundleChecksum}`).join(","),
    ],
    queryFn: async () => {
      return Promise.all(
        readyPlugins.map(async (status) => {
          try {
            const module = await loadPluginModule(status.pluginId, status.bundleChecksum);
            return module.PLUGIN_TRAY_FAMILIES as
              Record<string, UnifiedTrayContentComponent> | undefined;
          } catch (error) {
            console.error(`Failed to load tray families for plugin ${status.pluginId}:`, error);
            return undefined;
          }
        })
      );
    },
    enabled: readyPlugins.length > 0,
  });

  return Object.assign({}, ...trayFamilies.filter(Boolean));
};

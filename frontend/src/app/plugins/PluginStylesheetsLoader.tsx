import { FC, useEffect } from "react";
import { useGetInstalledPlugins } from "../marketplace/hooks/useGetInstalledPlugins";
import { getStylesheets } from "./hooks/registry/stylesheet/pluginStylesheetRegistry";
import { ensurePluginStylesheet } from "./utils/ensurePluginStylesheet";
import { loadPluginModule } from "./utils/loadPluginModule";

/**
 * Loads each installed plugin's app-wide stylesheet(s), registered via
 * appWideAPI.registerStylesheets, exactly once — independent of whether the
 * user has ever navigated to a cluster or to that plugin's resource view.
 * Mounted once at the app root (see App.tsx), unlike PluginResourceView's
 * per-view `stylesheet` field, which only loads when that specific view
 * mounts inside a connected cluster.
 *
 * Dynamically importing a plugin bundle that's already been loaded (e.g. by
 * PluginResourceView) is a cache hit — the browser's module registry dedupes
 * import() calls by URL, so this doesn't re-run the plugin's module-level
 * registration code or refetch the bundle.
 */
export const PluginStylesheetsLoader: FC = () => {
  const { readyPlugins } = useGetInstalledPlugins();

  useEffect(() => {
    for (const plugin of readyPlugins) {
      loadPluginModule(plugin.pluginId, plugin.bundleChecksum)
        .then(() => ensurePluginStylesheet(plugin.pluginId, getStylesheets(plugin.pluginId)))
        .catch((err) => {
          console.error(`Failed to load stylesheets for plugin ${plugin.pluginId}:`, err);
        });
    }
  }, [readyPlugins]);

  return null;
};

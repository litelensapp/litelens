import { FC, lazy, Suspense, useMemo } from "react";
import { PluginNotInstalledEmptyState } from "../../marketplace/components/PluginNotInstalledEmptyState";
import { useGetInstalledPlugin } from "./hooks/useGetInstalledPlugin";
import { PluginCrashedError } from "./components/PluginCrashedError";
import { PluginDisabledEmptyState } from "./components/PluginDisabledEmptyState";
import { PluginErrorBoundary } from "./components/PluginErrorBoundary";
import { PluginLoadingFallback } from "./components/PluginLoadingFallback";
import { ensurePluginStylesheet } from "./utils/ensurePluginStylesheet";

/**
 * Props passed to the dynamically-imported plugin's PluginView component.
 * Must be kept in sync with the plugin's export by hand, since the plugin bundle
 * is loaded at runtime via import() and cannot be statically type-checked. Plugins
 * source all cluster-scoped capabilities themselves via @litelens/core's
 * useClusterWideAPI(), so this component takes no props today.
 */
type PluginViewProps = Record<string, never>;

interface PluginResourceViewProps {
  pluginId: string;
  pluginName: string;
  /**
   * Whether the currently active resource belongs to this plugin. A READY
   * plugin's view stays mounted (just visually hidden) even when inactive —
   * that's how it registers its own nav entry (via useRegisterNavEntry inside
   * the plugin's own PluginView) before the user has ever navigated to it.
   */
  isActive: boolean;
  onGoToMarketplace: () => void;
}

export const PluginResourceView: FC<PluginResourceViewProps> = ({
  pluginId,
  pluginName,
  isActive,
  onGoToMarketplace,
}) => {
  const { status: pluginStatus, bundleChecksum } = useGetInstalledPlugin(pluginId, {
    hasAttemptedInstall: true,
  });

  // Build dynamic import URL with cache-busting checksum from plugin status
  // The checksum changes when the plugin is reinstalled/updated, invalidating browser cache
  const cacheVersion =
    pluginStatus === "READY" && bundleChecksum ? bundleChecksum.substring(0, 8) : "unknown";
  const pluginAssetUrl = `/api/plugins/${pluginId}/dist/index.js?v=${cacheVersion}`;

  // Memoized on the cache-busted URL so polling doesn't create a new lazy
  // component and remount/re-import the plugin on every tick.
  const PluginViewDynamic = useMemo(
    () =>
      lazy(() =>
        // eslint-disable-next-line @typescript-eslint/no-explicit-any
        import(/* @vite-ignore */ pluginAssetUrl as any).then((m: any) => {
          ensurePluginStylesheet(pluginId, m.PLUGIN_STYLES);
          return { default: m.PluginView as FC<PluginViewProps> };
        })
      ),
    [pluginAssetUrl, pluginId]
  );

  // READY plugins mount unconditionally (kept alive even while another
  // resource is active) so the plugin's own PluginView can call
  // useRegisterNavEntry() and push its sidebar entry into the host before
  // the user has ever navigated to it — no separate host-known contract
  // needed beyond the existing PLUGIN_VIEW/PLUGIN_STYLES exports.
  if (pluginStatus === "READY") {
    return (
      <PluginErrorBoundary onGoToMarketplace={onGoToMarketplace}>
        <Suspense fallback={isActive ? <PluginLoadingFallback /> : null}>
          {/* display:contents when active so this wrapper generates no box —
              plugin views rely on h-full/flex-1 resolving against the host's
              <main>, which only works if they're direct box-model children
              of it. hidden still fully unmounts-from-layout (but keeps
              mounted in React) when inactive. */}
          <div className={isActive ? "contents" : "hidden"}>
            {/* PluginViewDynamic is memoized on pluginAssetUrl (see above),
                not recreated on every render — safe despite the static-components rule. */}
            {/* eslint-disable-next-line react-hooks/static-components */}
            <PluginViewDynamic />
          </div>
        </Suspense>
      </PluginErrorBoundary>
    );
  }

  // Non-READY states only matter as full-page takeovers when the user is
  // actually looking at this plugin's resource — otherwise nothing to show.
  if (!isActive) {
    return null;
  }

  if (pluginStatus === "INSTALLING") {
    return <PluginLoadingFallback />;
  }

  if (pluginStatus === "NOT_INSTALLED") {
    return (
      <PluginNotInstalledEmptyState pluginName={pluginName} onGoToMarketplace={onGoToMarketplace} />
    );
  }

  if (pluginStatus === "DISABLED") {
    return <PluginDisabledEmptyState onGoToMarketplace={onGoToMarketplace} />;
  }

  // CRASHED || INCOMPATIBLE
  return <PluginCrashedError onGoToMarketplace={onGoToMarketplace} />;
};

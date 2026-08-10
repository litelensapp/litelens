import { SharedNamespaceContext, SharedUnifiedTrayContext } from "@litelens/design-system";
import { FC, lazy, Suspense, useMemo } from "react";
import { PluginNotInstalledEmptyState } from "../../marketplace/components/PluginNotInstalledEmptyState";
import { useGetInstalledPlugin } from "./hooks/useGetInstalledPlugin";
import { useGetNamespaces } from "../modules/base/namespaces/hooks/data-access/useGetNamespaces";
import { useDetailDrawerContext } from "../shared/components/details/DetailDrawerContext";
import { useUnifiedTray } from "../shared/components/trays/unified/UnifiedTrayContext";
import { useResourceLinks } from "../shared/hooks/useResourceLinks";
import { PluginCrashedError } from "./components/PluginCrashedError";
import { PluginErrorBoundary } from "./components/PluginErrorBoundary";
import { PluginLoadingFallback } from "./components/PluginLoadingFallback";
import { ensurePluginStylesheet } from "./utils/ensurePluginStylesheet";

/**
 * Props passed to the dynamically-imported plugin's PluginView component.
 * Must be kept in sync with the plugin's export by hand, since the plugin bundle
 * is loaded at runtime via import() and cannot be statically type-checked.
 * Shared boundary types (SharedNamespaceContext, SharedUnifiedTrayContext)
 * live in the design system so both sides use one source of truth.
 */
interface PluginViewProps {
  activeResource: string;
  activeContext: string;
  namespace: string;
  onNavigateToView: (view: string) => void;
  onToggleNamespaceDetail: (name?: string) => void;
  namespaces: SharedNamespaceContext[];
  unifiedTray: SharedUnifiedTrayContext | null;
  getResourceLinks: (resource: {
    kind: string;
    name: string;
    namespace?: string;
  }) => Array<{ label: string; href: string }>;
}

interface PluginResourceViewProps {
  pluginId: string;
  pluginName: string;
  viewType: string;
  activeContext: string;
  namespace: string;
  onNavigateToView: (view: string) => void;
  onGoToMarketplace: () => void;
}

export const PluginResourceView: FC<PluginResourceViewProps> = ({
  pluginId,
  pluginName,
  viewType,
  activeContext,
  namespace,
  onNavigateToView,
  onGoToMarketplace,
}) => {
  const { onToggleNamespaceDetail } = useDetailDrawerContext();
  const unifiedTray = useUnifiedTray();
  const resourceLinksMap = useResourceLinks();
  const { data: namespaces = [] } = useGetNamespaces(activeContext);

  // This view is only reached for a plugin the user has already navigated to,
  // so the real CRASHED/INCOMPATIBLE status must always surface (never masked
  // as NOT_INSTALLED the way a fresh marketplace visitor's would be).
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

  const getResourceLinks = (resource: { kind: string; name: string; namespace?: string }) => {
    const kind = resource.kind.toLowerCase();
    const linkFn = resourceLinksMap[kind];
    if (linkFn && resource.namespace) {
      linkFn(resource.namespace, resource.name);
      return [{ label: `View ${resource.kind}`, href: "#" }];
    }
    return [];
  };

  // Route plugin status to appropriate UI
  if (pluginStatus === "INSTALLING") {
    return <PluginLoadingFallback />;
  }

  if (pluginStatus === "NOT_INSTALLED") {
    return (
      <PluginNotInstalledEmptyState pluginName={pluginName} onGoToMarketplace={onGoToMarketplace} />
    );
  }

  if (pluginStatus === "CRASHED" || pluginStatus === "INCOMPATIBLE") {
    return <PluginCrashedError onGoToMarketplace={onGoToMarketplace} />;
  }

  // READY state — render dynamic import
  return (
    <PluginErrorBoundary onGoToMarketplace={onGoToMarketplace}>
      <Suspense fallback={<PluginLoadingFallback />}>
        {/* PluginViewDynamic is memoized on pluginAssetUrl (see above),
            not recreated on every render — safe despite the static-components rule. */}
        {/* eslint-disable-next-line react-hooks/static-components */}
        <PluginViewDynamic
          activeResource={viewType}
          activeContext={activeContext}
          namespace={namespace}
          onNavigateToView={onNavigateToView}
          onToggleNamespaceDetail={onToggleNamespaceDetail}
          namespaces={namespaces}
          unifiedTray={unifiedTray}
          getResourceLinks={getResourceLinks}
        />
      </Suspense>
    </PluginErrorBoundary>
  );
};

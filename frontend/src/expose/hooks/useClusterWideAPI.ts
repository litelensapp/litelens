import { useMainLayoutContext } from "../../app/clusters/MainLayoutContext";
import { useGetNamespaces } from "../../app/clusters/modules/base/namespaces/hooks/data-access/useGetNamespaces";
import { useUnifiedTray } from "../../app/clusters/shared/components/trays/unified/UnifiedTrayContext";
import { useResourceLinks } from "../../app/clusters/shared/hooks/useResourceLinks";
import { useRegisterClusterWideEvents } from "../../app/clusters/shared/hooks/registry/event/useRegisterClusterWideEvents";
import { useRegisterNavEntry } from "../../app/clusters/plugins/hooks/registry/nav/useRegisterNavEntry";
import { useRegisterTrayFamilies } from "../../app/clusters/plugins/hooks/registry/tray/useRegisterTrayFamilies";

/**
 * Cluster-scoped API surface exposed to plugins. Valid only for components
 * rendered inside MainLayout's subtree (MainLayoutProvider → DetailDrawerProvider),
 * i.e. anywhere within the single-cluster view. A future useAppWideAPI() will cover
 * capabilities valid across the whole app, outside any single cluster's scope.
 */
export function useClusterWideAPI() {
  const {
    activeContext,
    activeResource,
    namespaces: activeNamespaces,
    onNavigateToView,
  } = useMainLayoutContext();

  const { data: availableNamespaces = [] } = useGetNamespaces(activeContext);
  const resourceLinks = useResourceLinks();
  const unifiedTray = useUnifiedTray();

  return {
    activeContext,
    activeNamespaces,
    activeResource,
    availableNamespaces,
    onNavigateToView,
    resourceLinks,
    unifiedTray,
    useRegisterClusterWideEvents,
    useRegisterNavEntry,
    useRegisterTrayFamilies,
  };
}

import { ErrorBoundary, renderErrorToast } from "@litelens/design-system";
import { NavEntry, NavItem } from "@litelens/core";
import { FC, lazy, Suspense, useEffect, useMemo, useRef, useState } from "react";
import { useCatchForbiddenResources } from "../shared/hooks/async-events/useCatchForbiddenResources";
import { useClusterWideEventListener } from "./shared/hooks/registry/event/useClusterWideEventListener";
import { MainLayoutProvider } from "./MainLayoutContext";
import { useGetNamespaceNames } from "./modules/base/namespaces/hooks/data-access/useGetNamespaceNames";
import { RESOURCE_LABEL, ViewType } from "./navConfig";
import { NavSidebar } from "./NavSidebar";
import { usePluginNavEntries } from "./plugins/hooks/registry/nav/usePluginNavEntries";
import { useGetInstalledPlugins } from "../marketplace/hooks/useGetInstalledPlugins";
import { usePluginTrayFamilies } from "./plugins/hooks/registry/tray/usePluginTrayFamilies";
import { PluginResourceView } from "./plugins/PluginResourceView";
import { DetailBlock } from "./shared/components/details/DetailBlock";
import { NamespaceMultiSelect } from "./shared/components/NamespaceMultiSelect";
import { UnifiedTrayOutlet } from "./shared/components/trays/unified/UnifiedTrayOutlet";
import { unifiedTrayRegistry } from "./shared/components/trays/unified/unifiedTrayRegistry";
import { useGetDefaultNamespaces } from "./modules/base/namespaces/hooks/data-access/useGetDefaultNamespaces";

const PodsView = lazy(() =>
  import("./modules/workloads/pods/PodsView").then((m) => ({
    default: m.PodsView,
  }))
);
const DeploymentsView = lazy(() =>
  import("./modules/workloads/deployments/DeploymentsView").then((m) => ({
    default: m.DeploymentsView,
  }))
);
const ServicesView = lazy(() =>
  import("./modules/networks/services/ServicesView").then((m) => ({
    default: m.ServicesView,
  }))
);
const NodesView = lazy(() =>
  import("./modules/base/nodes/NodesView").then((m) => ({
    default: m.NodesView,
  }))
);
const NamespacesView = lazy(() =>
  import("./modules/base/namespaces/NamespacesView").then((m) => ({
    default: m.NamespacesView,
  }))
);
const DaemonSetsView = lazy(() =>
  import("./modules/workloads/daemonsets/DaemonSetsView").then((m) => ({
    default: m.DaemonSetsView,
  }))
);
const ReplicaSetsView = lazy(() =>
  import("./modules/workloads/replicasets/ReplicaSetsView").then((m) => ({
    default: m.ReplicaSetsView,
  }))
);
const ConfigMapsView = lazy(() =>
  import("./modules/configs/configmaps/ConfigMapsView").then((m) => ({
    default: m.ConfigMapsView,
  }))
);
const EndpointSlicesView = lazy(() =>
  import("./modules/networks/endpointslices/EndpointSlicesView").then((m) => ({
    default: m.EndpointSlicesView,
  }))
);
const EndpointsView = lazy(() =>
  import("./modules/networks/endpoints/EndpointsView").then((m) => ({
    default: m.EndpointsView,
  }))
);
const StatefulSetsView = lazy(() =>
  import("./modules/workloads/statefulsets/StatefulSetsView").then((m) => ({
    default: m.StatefulSetsView,
  }))
);
const JobsView = lazy(() =>
  import("./modules/workloads/jobs/JobsView").then((m) => ({
    default: m.JobsView,
  }))
);
const CronJobsView = lazy(() =>
  import("./modules/workloads/cronjobs/CronJobsView").then((m) => ({
    default: m.CronJobsView,
  }))
);
const SecretsView = lazy(() =>
  import("./modules/configs/secrets/SecretsView").then((m) => ({
    default: m.SecretsView,
  }))
);
const ResourceQuotasView = lazy(() =>
  import("./modules/configs/resourcequotas/ResourceQuotasView").then((m) => ({
    default: m.ResourceQuotasView,
  }))
);
const LimitRangesView = lazy(() =>
  import("./modules/configs/limitranges/LimitRangesView").then((m) => ({
    default: m.LimitRangesView,
  }))
);
const HPAView = lazy(() =>
  import("./modules/configs/hpas/HPAView").then((m) => ({
    default: m.HPAView,
  }))
);
const PodDisruptionBudgetsView = lazy(() =>
  import("./modules/configs/pdbs/PodDisruptionBudgetsView").then((m) => ({
    default: m.PodDisruptionBudgetsView,
  }))
);
const ValidatingWebhookConfigsView = lazy(() =>
  import("./modules/configs/validatingwebhookconfigs/ValidatingWebhookConfigsView").then((m) => ({
    default: m.ValidatingWebhookConfigsView,
  }))
);
const IngressesView = lazy(() =>
  import("./modules/networks/ingresses/IngressesView").then((m) => ({
    default: m.IngressesView,
  }))
);
const IngressClassesView = lazy(() =>
  import("./modules/networks/ingressclasses/IngressClassesView").then((m) => ({
    default: m.IngressClassesView,
  }))
);
const NetworkPoliciesView = lazy(() =>
  import("./modules/networks/networkpolicies/NetworkPoliciesView").then((m) => ({
    default: m.NetworkPoliciesView,
  }))
);
const PortForwardingView = lazy(() =>
  import("./modules/networks/portforwarding/PortForwardingView").then((m) => ({
    default: m.PortForwardingView,
  }))
);
const PersistentVolumeClaimsView = lazy(() =>
  import("./modules/storages/pvcs/PersistentVolumeClaimsView").then((m) => ({
    default: m.PersistentVolumeClaimsView,
  }))
);
const PersistentVolumesView = lazy(() =>
  import("./modules/storages/pvs/PersistentVolumesView").then((m) => ({
    default: m.PersistentVolumesView,
  }))
);
const ServiceAccountsView = lazy(() =>
  import("./modules/accessControls/serviceaccounts/ServiceAccountsView").then((m) => ({
    default: m.ServiceAccountsView,
  }))
);
const ClusterRolesView = lazy(() =>
  import("./modules/accessControls/clusterroles/ClusterRolesView").then((m) => ({
    default: m.ClusterRolesView,
  }))
);
const RolesView = lazy(() =>
  import("./modules/accessControls/roles/RolesView").then((m) => ({
    default: m.RolesView,
  }))
);
const ClusterRoleBindingsView = lazy(() =>
  import("./modules/accessControls/clusterrolebindings/ClusterRoleBindingsView").then((m) => ({
    default: m.ClusterRoleBindingsView,
  }))
);
const RoleBindingsView = lazy(() =>
  import("./modules/accessControls/rolebindings/RoleBindingsView").then((m) => ({
    default: m.RoleBindingsView,
  }))
);
const StorageClassesView = lazy(() =>
  import("./modules/storages/storageclasses/StorageClassesView").then((m) => ({
    default: m.StorageClassesView,
  }))
);
const EventsView = lazy(() =>
  import("./modules/base/events/EventsView").then((m) => ({
    default: m.EventsView,
  }))
);
const PriorityClassesView = lazy(() =>
  import("./modules/configs/priorityclasses/PriorityClassesView").then((m) => ({
    default: m.PriorityClassesView,
  }))
);
const LeasesView = lazy(() =>
  import("./modules/configs/leases/LeasesView").then((m) => ({
    default: m.LeasesView,
  }))
);
const OverviewView = lazy(() =>
  import("./modules/overview/OverviewView").then((m) => ({
    default: m.OverviewView,
  }))
);
interface MainLayoutProps {
  activeContext: string;
  onOpenMarketplace: () => void;
}

export const MainLayout: FC<MainLayoutProps> = ({ activeContext, onOpenMarketplace }) => {
  // Listen for plugin:event emissions and route to registered handlers
  useClusterWideEventListener();

  const [activeResource, setActiveResource] = useState<ViewType>("overview");
  const [openGroups, setOpenGroups] = useState<Set<string>>(
    () => new Set(["workloads", "network", "config", "storage", "access-control"])
  );

  const [namespaces, setNamespaces] = useState<string[]>([]);
  const { data: namespaceNames = [] } = useGetNamespaceNames(activeContext);
  const { data: defaultNamespaces } = useGetDefaultNamespaces(activeContext);
  // Tracks the last-applied "<context>:<defaults>" snapshot, so the seed below
  // re-applies both on first load for a context AND whenever the persisted
  // defaults change (e.g. the user saves new ones in cluster settings), while
  // not re-running on every render or clobbering in-session filter changes
  // made via handleNamespacesChange in between saves.
  const [appliedDefaultKey, setAppliedDefaultKey] = useState<string | null>(null);

  // Settings-persisted default namespaces are the source of truth for the
  // filter, applied on load and re-applied whenever they're saved again. In
  // between saves, in-session filter changes are kept in memory only (notxw
  // persisted) — reloading or restarting the app always rolls back to the
  // latest saved default. Defaults naming a namespace that no longer exists on
  // the cluster are dropped.
  if (defaultNamespaces && namespaceNames.length > 0) {
    const defaultKey = `${activeContext}:${JSON.stringify(defaultNamespaces)}`;
    if (appliedDefaultKey !== defaultKey) {
      setAppliedDefaultKey(defaultKey);
      const existing = new Set(namespaceNames);
      setNamespaces(defaultNamespaces.filter((ns) => existing.has(ns)));
    }
  }

  // Union with the persisted defaults so a namespace typed manually in cluster
  // settings (e.g. because the cluster can't be listed due to RBAC) shows up as
  // a selectable filter option immediately, without switching clusters to
  // remount and re-fetch.
  const sortedNamespaceNames = useMemo(
    () => Array.from(new Set([...namespaceNames, ...(defaultNamespaces ?? [])])).sort(),
    [namespaceNames, defaultNamespaces]
  );

  // Nav data comes from usePluginNavEntries reading pluginNavRegistry, which
  // each plugin's own PluginView populates via useRegisterNavEntry() on mount
  // (below, in JSX every non-uninstalled plugin is always mounted, hidden
  // when inactive — see PluginResourceView) — the plugin pushes its own nav
  // entry rather than the host reading a static export or importing any
  // plugin-specific nav contract.
  const pluginNavData = usePluginNavEntries();
  // Plugin groups registered with defaultOpen expand the first time they're
  // seen (e.g. right after install). Tracked separately from openGroups so a
  // user collapsing the group afterwards isn't overridden on re-registration.
  const seededDefaultOpenGroups = useRef(new Set<string>());
  const pluginNavEntries = pluginNavData.navEntries;
  useEffect(() => {
    const toSeed = pluginNavEntries
      .filter(
        (entry): entry is Extract<NavEntry<string>, { kind: "group" }> => entry.kind === "group"
      )
      .filter(
        (entry) => entry.group.defaultOpen && !seededDefaultOpenGroups.current.has(entry.group.id)
      )
      .map((entry) => entry.group.id);

    if (toSeed.length === 0) return;

    for (const id of toSeed) {
      seededDefaultOpenGroups.current.add(id);
    }
    setOpenGroups((prev) => new Set([...prev, ...toSeed]));
  }, [pluginNavEntries]);
  const { pluginStatuses } = useGetInstalledPlugins();
  const mountedPlugins = useMemo(
    () => pluginStatuses.filter((s) => s.status !== "NOT_INSTALLED"),
    [pluginStatuses]
  );
  // Tray families come from usePluginTrayFamilies reading pluginTrayRegistry,
  // which each plugin's own PluginView populates via useRegisterTrayFamilies()
  // on mount (mirrors the nav-entry mechanism above) — the plugin pushes its
  // own tray-family components rather than the host reading a static export.
  const pluginTrayRegistry = usePluginTrayFamilies();
  const mergedResourceLabels = useMemo(
    () => ({ ...RESOURCE_LABEL, ...pluginNavData.resourceLabels }),
    [pluginNavData.resourceLabels]
  );
  const mergedTrayRegistry = useMemo(
    () => ({ ...unifiedTrayRegistry, ...pluginTrayRegistry }),
    [pluginTrayRegistry]
  );

  const { forbiddenResources } = useCatchForbiddenResources(activeResource, {
    labelMap: mergedResourceLabels,
    activeContext,
  });

  function handleNamespacesChange(ns: string[]) {
    setNamespaces(ns);
  }

  function toggleGroup(id: string) {
    setOpenGroups((prev) => {
      const next = new Set(prev);
      if (next.has(id)) next.delete(id);
      else next.add(id);
      return next;
    });
  }

  function handleSelectItem(item: NavItem<ViewType>) {
    if (item.view) {
      setActiveResource(item.view as ViewType);
      if (forbiddenResources.has(item.view)) {
        const label =
          mergedResourceLabels[item.view as keyof typeof mergedResourceLabels] ?? item.view;
        renderErrorToast({ title: `Access denied: cannot list ${label}` });
      }
    }
  }

  return (
    <MainLayoutProvider
      activeContext={activeContext}
      activeResource={activeResource}
      namespaces={namespaces}
      onNamespacesChange={handleNamespacesChange}
      onNavigateToView={setActiveResource}
      className="flex h-full min-w-0 flex-1 overflow-hidden"
    >
      {/* Sidebar */}
      <NavSidebar
        activeResource={activeResource}
        openGroups={openGroups}
        onToggleGroup={toggleGroup}
        onSelectItem={handleSelectItem}
        pluginNavEntries={pluginNavData.navEntries}
      />

      {/* Main area */}
      <div className="flex flex-1 flex-col overflow-hidden">
        {/* Top bar */}
        <header className="flex shrink-0 items-center justify-between gap-3 border-b px-4 py-2">
          <span className="text-h1 font-medium">{activeContext}</span>
          <NamespaceMultiSelect
            namespaces={namespaces}
            availableNamespaces={sortedNamespaceNames}
            onNamespacesChange={handleNamespacesChange}
          />
        </header>

        {/* Content */}
        <main className="flex-1 overflow-auto p-4">
          <ErrorBoundary>
            <Suspense fallback={<div className="text-muted-foreground p-4 text-sm">Loading…</div>}>
              {activeResource === "overview" && (
                <OverviewView onNavigateToView={setActiveResource} />
              )}
              {activeResource === "pods" && <PodsView />}
              {activeResource === "deployments" && <DeploymentsView />}
              {activeResource === "daemonsets" && <DaemonSetsView />}
              {activeResource === "statefulsets" && <StatefulSetsView />}
              {activeResource === "jobs" && <JobsView />}
              {activeResource === "cronjobs" && <CronJobsView />}
              {activeResource === "replicasets" && <ReplicaSetsView />}
              {activeResource === "configmaps" && <ConfigMapsView />}
              {activeResource === "secrets" && <SecretsView />}
              {activeResource === "resourcequotas" && <ResourceQuotasView />}
              {activeResource === "limitranges" && <LimitRangesView />}
              {activeResource === "hpa" && <HPAView />}
              {activeResource === "pdbs" && <PodDisruptionBudgetsView />}
              {activeResource === "validatingwebhookconfigs" && <ValidatingWebhookConfigsView />}
              {activeResource === "ingresses" && <IngressesView />}
              {activeResource === "ingressclasses" && <IngressClassesView />}
              {activeResource === "networkpolicies" && <NetworkPoliciesView />}
              {activeResource === "portforwarding" && <PortForwardingView />}
              {activeResource === "pvcs" && <PersistentVolumeClaimsView />}
              {activeResource === "pvs" && <PersistentVolumesView />}
              {activeResource === "storageclasses" && <StorageClassesView />}
              {activeResource === "endpointslices" && <EndpointSlicesView />}
              {activeResource === "endpoints" && <EndpointsView />}
              {activeResource === "services" && <ServicesView />}
              {activeResource === "nodes" && <NodesView />}
              {activeResource === "namespaces" && <NamespacesView />}
              {activeResource === "serviceaccounts" && <ServiceAccountsView />}
              {activeResource === "clusterroles" && <ClusterRolesView />}
              {activeResource === "roles" && <RolesView />}
              {activeResource === "clusterrolebindings" && <ClusterRoleBindingsView />}
              {activeResource === "rolebindings" && <RoleBindingsView />}
              {activeResource === "priorityclasses" && <PriorityClassesView key={activeContext} />}
              {activeResource === "leases" && <LeasesView />}
              {activeResource === "events" && <EventsView />}

              {/* Every non-uninstalled plugin stays mounted (hidden when inactive)
                  so a READY plugin's own PluginView can register its nav entry
                  before the user has navigated to it — see PluginResourceView. */}
              {mountedPlugins.map((status) => (
                <PluginResourceView
                  key={status.pluginId}
                  pluginId={status.pluginId}
                  pluginName={status.name}
                  isActive={pluginNavData.viewTypeToPluginId[activeResource] === status.pluginId}
                  onGoToMarketplace={onOpenMarketplace}
                />
              ))}
            </Suspense>

            <DetailBlock onNavigateToPortForwarding={() => setActiveResource("portforwarding")} />

            <UnifiedTrayOutlet registry={mergedTrayRegistry} />
          </ErrorBoundary>
        </main>
      </div>
    </MainLayoutProvider>
  );
};

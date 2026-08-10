import { Button, Divider, FrownIcon, LoadingSpinner } from "@litelens/design-system";
import type { dto } from "@wailsjs/go/models";
import { FC, useCallback, useEffect, useMemo, useRef, useState } from "react";
import { maskTerminalStatus } from "../shared/utils/maskTerminalStatus";
import { useGetVersion } from "../updater/hooks/data-access/useGetVersion";
import { PluginCard } from "./components/PluginCard";
import { PluginCardFallback } from "./components/PluginCardFallback";
import {
  toastPluginInstallFailed,
  toastPluginInstallSucceeded,
  toastPluginRemovalFailed,
  toastPluginRemovalSucceeded,
} from "./components/PluginToasts";
import { useGetHostPlatform } from "./hooks/useGetHostPlatform";
import { useGetInstalledPlugins } from "./hooks/useGetInstalledPlugins";
import { PluginManifest, useGetPluginsFromMarketplace } from "./hooks/useGetPluginsFromMarketplace";
import { useMutateInstallPlugin } from "./hooks/useMutateInstallPlugin";
import { useMutateRemovePlugin } from "./hooks/useMutateRemovePlugin";

// Backend placeholder for a plugin installed before checksum-tracking existed
// (or whose checksum file is otherwise missing) — never a real sha256, so it
// must not be treated as "matches the marketplace version".
const PLACEHOLDER_BUNDLE_CHECKSUM =
  "0000000000000000000000000000000000000000000000000000000000000000";

export const MarketplaceView: FC<{
  onGoToMarketplaceSettings?: () => void;
}> = ({ onGoToMarketplaceSettings }) => {
  const {
    data: marketplacePlugins = [],
    isLoading: isLoadingMarketplace,
    isError: isMarketplaceError,
    error: marketplaceError,
  } = useGetPluginsFromMarketplace();

  // Get current app version for compatibility checking
  const { data: hostVersion = "0.1.0" } = useGetVersion();

  // Get current OS ("linux" | "darwin" | "windows") to highlight the matching compatibility chip
  const { data: hostPlatform } = useGetHostPlatform();

  // Fetch all installed plugins (plural) for joining with marketplace plugins
  const { pluginStatuses: installedPlugins = [], isLoading: isLoadingInstalled } =
    useGetInstalledPlugins();

  // Mutation hooks for install and remove (call-time arguments, not hook-time)
  const installMutation = useMutateInstallPlugin();
  const removeMutation = useMutateRemovePlugin();

  // Track per-plugin "attempted install this mount" as a Set in state
  // Used to mask CRASHED/INCOMPATIBLE statuses as NOT_INSTALLED for display
  // (replaces the old single hasAttemptedThisMount boolean)
  const [attemptedInstalls, setAttemptedInstalls] = useState<Set<string>>(new Set());

  // Track per-plugin IDs currently being removed to fix the concurrency issue.
  // useMutation's .isPending and .variables are shared across all calls on the
  // same hook instance. This Set lets us track in-flight removes independently
  // per pluginId instead of relying on mutation.variables which gets overwritten.
  const [removingIds, setRemovingIds] = useState<Set<string>>(new Set());

  // Generic install handler called from card's onInstall/onUpdate callbacks
  const handleInstall = useCallback(
    async (pluginId: string, pluginName: string, targetTag?: string) => {
      setAttemptedInstalls((prev) => new Set(prev).add(pluginId));
      try {
        // Look up the manifest to get its sourceUrl
        const manifest = marketplacePlugins.find((p) => p.id === pluginId);
        const sourceUrl = manifest?.sourceUrl ?? "";
        await installMutation.mutateAsync({ pluginId, targetTag, sourceUrl });
      } catch (error) {
        console.error(`Failed to install ${pluginName} plugin:`, error);
        toastPluginInstallFailed(pluginName, error);
      }
    },
    [installMutation, marketplacePlugins]
  );

  // Generic remove handler called from card's onRemove callback
  const handleRemove = useCallback(
    async (pluginId: string, pluginName: string) => {
      setRemovingIds((prev) => new Set(prev).add(pluginId));
      try {
        await removeMutation.mutateAsync({ pluginId });
        toastPluginRemovalSucceeded(pluginName);
      } catch (error) {
        console.error(`Failed to remove ${pluginName} plugin:`, error);
        toastPluginRemovalFailed(pluginName, error);
      } finally {
        setRemovingIds((prev) => {
          const updated = new Set(prev);
          updated.delete(pluginId);
          return updated;
        });
      }
    },
    [removeMutation]
  );

  // Track previous statuses per plugin ID for toast on install completion
  const previousStatusesRef = useRef<Record<string, string>>({});

  useEffect(() => {
    const marketplacePluginsById = new Map(marketplacePlugins.map((p) => [p.id, p]));

    for (const status of installedPlugins) {
      const previousStatus = previousStatusesRef.current[status.pluginId];
      previousStatusesRef.current[status.pluginId] = status.status;

      if (previousStatus !== "INSTALLING") continue;

      if (status.status === "READY") {
        // Find plugin name from marketplace for toast
        const plugin = marketplacePluginsById.get(status.pluginId);
        toastPluginInstallSucceeded(plugin?.name ?? status.pluginId);
      } else if (status.status === "CRASHED" || status.status === "INCOMPATIBLE") {
        const plugin = marketplacePluginsById.get(status.pluginId);
        toastPluginInstallFailed(plugin?.name ?? status.pluginId, status.error);
      }
    }
  }, [installedPlugins, marketplacePlugins]);

  // Build installed/available sections with deduplication.
  // "Installed" is reserved for plugins that are actually on disk (READY /
  // CRASHED / INCOMPATIBLE) *and* have a resolvable manifest — INSTALLING is
  // still in progress, and a manifestless entry (e.g. crashed before its
  // .plugin-metadata.json was ever written, or metadata corrupted, with no
  // matching marketplace listing to fall back on) can't render a real
  // PluginCard either way. Both cases are pushed to "Available" instead,
  // where the existing manifest-optional rendering already covers them:
  // a live progress bar when the manifest is known, PluginCardFallback when
  // it isn't. This keeps manifest non-optional for everything in "installed".
  const { installed, available } = useMemo(() => {
    const marketplacePluginsById = new Map(marketplacePlugins.map((p) => [p.id, p]));
    // Plugins that belong in "Available" beyond the plain marketplace catalog:
    // still installing, or installed-but-manifestless. Keyed by pluginId.
    const extraByPluginId = new Map<string, dto.InstalledPlugin>();

    const installed: Array<{
      installedPlugin: dto.InstalledPlugin;
      manifest: dto.InstalledPlugin | PluginManifest;
    }> = [];

    for (const installedPlugin of installedPlugins) {
      const hasAttempted = attemptedInstalls.has(installedPlugin.pluginId);
      const displayStatus = maskTerminalStatus(installedPlugin.status, hasAttempted);
      if (displayStatus === "NOT_INSTALLED") continue;

      if (displayStatus === "INSTALLING") {
        extraByPluginId.set(installedPlugin.pluginId, installedPlugin);
        continue;
      }

      const marketplaceManifest = marketplacePluginsById.get(installedPlugin.pluginId);
      const manifest = marketplaceManifest ?? (installedPlugin.id ? installedPlugin : undefined);
      if (manifest) installed.push({ installedPlugin, manifest });
      else extraByPluginId.set(installedPlugin.pluginId, installedPlugin);
    }

    const installedIds = new Set(installed.map(({ installedPlugin }) => installedPlugin.pluginId));

    const available: Array<{
      id: string;
      manifest: PluginManifest | undefined;
      installStatus: dto.InstalledPlugin | undefined;
    }> = marketplacePlugins
      .filter((p) => !installedIds.has(p.id))
      .map((manifest) => ({
        id: manifest.id,
        manifest,
        installStatus: extraByPluginId.get(manifest.id),
      }));

    // Entries with no matching marketplace listing (catalog changed mid-install,
    // or a crashed/corrupted install the catalog never covered) still need to
    // render somewhere — surface them via PluginCardFallback.
    for (const [pluginId, installStatus] of extraByPluginId) {
      if (!marketplacePluginsById.has(pluginId) && !installedIds.has(pluginId)) {
        available.push({ id: pluginId, manifest: undefined, installStatus });
      }
    }

    return { installed, available };
  }, [installedPlugins, attemptedInstalls, marketplacePlugins]);

  return (
    <div className="flex min-h-0 flex-1 flex-col overflow-hidden">
      <header className="flex h-14 shrink-0 items-center justify-between border-b px-6 py-3">
        <h2 className="text-sm font-semibold">Marketplace</h2>
      </header>
      <div className="flex-1 overflow-auto px-6 py-4">
        {isMarketplaceError ? (
          <div className="flex flex-col gap-4 py-16 text-center">
            <div>
              <FrownIcon className="text-muted-foreground mx-auto mb-3 size-12" />
              <p className="text-destructive text-lg font-medium">Couldn&apos;t load marketplace</p>
              <p className="text-muted-foreground mt-2 text-xs">
                {marketplaceError?.message || "Check your Access Token in Settings"}
              </p>
            </div>
            {marketplaceError?.message && (
              <div>
                <p className="text-muted-foreground mt-2 text-xs">
                  Please check your Marketplace Repository URL
                </p>
                <Button className="mt-4" onClick={onGoToMarketplaceSettings}>
                  Go to Marketplace settings
                </Button>
              </div>
            )}
          </div>
        ) : (
          <div className="flex min-h-full flex-col gap-4">
            {/* Installed Plugins Section */}
            {installed.length > 0 && (
              <>
                <section className="flex flex-col gap-4">
                  <h3 className="text-sm font-semibold">
                    Installed Plugins{installed.length > 0 ? ` (${installed.length})` : ""}
                  </h3>
                  {isLoadingInstalled ? (
                    <div className="flex flex-col items-center justify-center gap-3 py-16">
                      <LoadingSpinner />
                      <p className="text-muted-foreground text-sm">Loading installed plugins...</p>
                    </div>
                  ) : (
                    <div className="flex flex-col gap-6">
                      {installed.map(({ installedPlugin, manifest }) => {
                        const hasAttempted = attemptedInstalls.has(installedPlugin.pluginId);
                        const displayStatus = maskTerminalStatus(
                          installedPlugin.status,
                          hasAttempted
                        );

                        const hasComparableChecksum = Boolean(
                          installedPlugin.bundleChecksum &&
                          installedPlugin.bundleChecksum !== "" &&
                          installedPlugin.bundleChecksum !== PLACEHOLDER_BUNDLE_CHECKSUM
                        );
                        const checksumMismatch =
                          hasComparableChecksum &&
                          installedPlugin.bundleChecksum !== manifest.bundle.sha256;
                        const versionMismatch =
                          (installedPlugin.installedVersion ?? "") !== manifest.version;
                        const updateAvailable: boolean =
                          displayStatus === "READY" && (checksumMismatch || versionMismatch);

                        const isPluginRemoving = removingIds.has(installedPlugin.pluginId);

                        return (
                          <PluginCard
                            key={installedPlugin.pluginId}
                            plugin={manifest}
                            hostVersion={hostVersion}
                            hostPlatform={hostPlatform}
                            installStatus={
                              displayStatus as
                                | "NOT_INSTALLED"
                                | "INSTALLING"
                                | "READY"
                                | "CRASHED"
                                | "INCOMPATIBLE"
                            }
                            installProgress={installedPlugin.progress}
                            isVerifying={displayStatus === "INSTALLING"}
                            updateAvailable={updateAvailable}
                            installedVersion={installedPlugin.installedVersion}
                            installedSize={installedPlugin.size}
                            onInstall={() => handleInstall(installedPlugin.pluginId, manifest.name)}
                            onUpdate={() => handleInstall(installedPlugin.pluginId, manifest.name)}
                            onRetry={() =>
                              handleInstall(
                                installedPlugin.pluginId,
                                manifest.name,
                                installedPlugin.installedVersion
                              )
                            }
                            onRemove={() => handleRemove(installedPlugin.pluginId, manifest.name)}
                            isRemoving={isPluginRemoving}
                          />
                        );
                      })}
                    </div>
                  )}
                </section>

                <Divider />
              </>
            )}

            {/* Available Plugins Section */}
            <section className="flex flex-1 flex-col gap-4">
              <h3 className="text-sm font-semibold">
                Available Plugins{available.length > 0 ? ` (${available.length})` : ""}
              </h3>
              {isLoadingMarketplace ? (
                <div className="flex flex-1 flex-col items-center justify-center gap-3 py-16">
                  <LoadingSpinner />
                  <p className="text-muted-foreground text-sm">Loading available plugins...</p>
                </div>
              ) : available.length === 0 ? (
                <div className="flex flex-1 flex-col items-center justify-center text-center">
                  <p className="text-muted-foreground">No available plugins</p>
                </div>
              ) : (
                <div className="flex flex-col gap-6">
                  {available.map(({ id, manifest, installStatus: installStatusEntry }) => {
                    if (!manifest) {
                      return (
                        <PluginCardFallback
                          key={id}
                          status={installStatusEntry!}
                          onRemove={() => handleRemove(id, id)}
                          isRemoving={removingIds.has(id)}
                        />
                      );
                    }

                    return (
                      <PluginCard
                        key={id}
                        plugin={manifest}
                        hostVersion={hostVersion}
                        hostPlatform={hostPlatform}
                        installStatus={installStatusEntry ? "INSTALLING" : "NOT_INSTALLED"}
                        installProgress={installStatusEntry?.progress ?? 0}
                        isVerifying={Boolean(installStatusEntry)}
                        updateAvailable={false}
                        installedVersion={undefined}
                        installedSize={undefined}
                        onInstall={() => handleInstall(id, manifest.name)}
                        onUpdate={undefined}
                        onRetry={undefined}
                        onRemove={undefined}
                        isRemoving={false}
                      />
                    );
                  })}
                </div>
              )}
            </section>
          </div>
        )}
      </div>
    </div>
  );
};

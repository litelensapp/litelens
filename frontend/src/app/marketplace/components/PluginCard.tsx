import {
  Badge,
  Button,
  cn,
  ConfirmationModal,
  Loader2Icon,
  Tooltip,
  TooltipContent,
  TooltipTrigger,
  Trash2Icon,
} from "@litelens/design-system";
import { FC, useMemo, useState } from "react";
import { PluginManifest } from "../hooks/useGetPluginsFromMarketplace";
import { formatBytes } from "../utils/formatBytes";
import { pluginLogoUrl } from "../utils/pluginLogoUrl";
import { compareVersions } from "../utils/semver";
import { DownloadProgressIndicator } from "./DownloadProgressIndicator";
import { PluginLogo } from "./PluginLogo";

// Keys match Manifest.os (Go GOOS values); labels are the human-readable names shown in chips.
const OS_CHIPS: { key: "linux" | "darwin" | "windows"; label: string }[] = [
  { key: "linux", label: "Linux" },
  { key: "darwin", label: "macOS" },
  { key: "windows", label: "Windows" },
];

interface PluginCardProps {
  plugin: PluginManifest;
  hostVersion: string;
  hostPlatform?: string; // "linux" | "darwin" | "windows" — highlights the matching compatibility chip
  installStatus: "NOT_INSTALLED" | "INSTALLING" | "READY" | "CRASHED" | "INCOMPATIBLE";
  installProgress?: number; // 0-100 for downloading
  isVerifying?: boolean;
  updateAvailable?: boolean;
  installedVersion?: string;
  installedSize?: number;
  onInstall?: () => void;
  onUpdate?: () => void;
  onRetry?: () => void;
  onRemove?: (pluginID: string) => void | Promise<void>;
  isRemoving?: boolean;
}

export const PluginCard: FC<PluginCardProps> = ({
  plugin,
  hostVersion,
  hostPlatform,
  installStatus,
  installProgress = 0,
  isVerifying = false,
  updateAvailable = false,
  installedVersion,
  installedSize,
  onInstall,
  onUpdate,
  onRetry,
  onRemove,
  isRemoving = false,
}) => {
  const [isRemoveDialogOpen, setIsRemoveDialogOpen] = useState(false);

  const isCompatible = useMemo(() => {
    // Dev builds report hostVersion="dev" (not semver) — skip the gate, it only applies to releases
    if (import.meta.env.DEV) return true;
    try {
      return (
        compareVersions(hostVersion, plugin.minimumHostVersion) >= 0 &&
        compareVersions(hostVersion, plugin.maximumHostVersion) <= 0
      );
    } catch {
      return false;
    }
  }, [hostVersion, plugin]);

  const isDisabled = !isCompatible;

  const handleRemoveClick = async () => {
    try {
      await onRemove?.(plugin.id);
      setIsRemoveDialogOpen(false);
    } catch (error) {
      // Error is logged in the hook, dialog stays open to let user retry
      console.error("Failed to remove plugin:", error);
    }
  };

  // Total download size (bundle + binary) — matches what actually lands on
  // disk, unlike plugin.bundle.size alone which is just the JS bundle asset.
  const marketplacePluginSize = formatBytes(plugin.bundle.size + plugin.binary.size);

  return (
    <div
      className={cn(
        "border-border bg-secondary-surface border-3 shadow-depth-1 transition-interactive hover:border-ring hover:shadow-depth-2 max-w-sm overflow-hidden rounded-lg",
        isDisabled && "pointer-events-none opacity-50"
      )}
    >
      {/* Header */}
      <div className="border-border border-b px-4 py-3">
        <div className="flex items-center justify-between gap-3">
          <div className="flex items-center gap-2">
            <PluginLogo
              src={pluginLogoUrl(plugin.id, plugin.assets?.logo)}
              alt={`${plugin.name} logo`}
            />
            <h3 className="text-h3 font-medium">{plugin.name}</h3>
          </div>
          <div className="flex items-center gap-2">
            <div className="flex gap-2">
              {updateAvailable && installStatus === "READY" && (
                <Tooltip>
                  <TooltipTrigger>
                    <Badge className="border border-blue-200 bg-blue-50 text-blue-900">
                      Update available
                    </Badge>
                  </TooltipTrigger>
                  <TooltipContent>
                    A newer version is available. Click Update to download.
                  </TooltipContent>
                </Tooltip>
              )}
              <Tooltip>
                <TooltipTrigger>
                  <Badge className="border border-amber-200 bg-amber-50 text-amber-900">
                    Unsigned
                  </Badge>
                </TooltipTrigger>
                <TooltipContent>
                  This plugin is not digitally signed. Install only if you trust the litelens
                  project.
                </TooltipContent>
              </Tooltip>
            </div>
            {(installStatus === "READY" || installStatus === "CRASHED") && (
              <Tooltip>
                <TooltipTrigger
                  render={
                    <Button
                      variant="ghost"
                      size="icon"
                      onClick={() => setIsRemoveDialogOpen(true)}
                      disabled={isRemoving}
                      aria-label="Remove plugin"
                    >
                      {isRemoving ? (
                        <Loader2Icon className="size-4 animate-spin" />
                      ) : (
                        <Trash2Icon className="size-4" />
                      )}
                    </Button>
                  }
                />
                <TooltipContent>Remove plugin</TooltipContent>
              </Tooltip>
            )}
          </div>
        </div>
      </div>

      {/* Description */}
      <div className="border-border border-b px-4 py-3">
        <p className="text-body text-muted-foreground">{plugin.description}</p>
      </div>

      {/* Metadata */}
      <div className="border-border border-b px-4 py-3">
        <div className="text-caption text-muted-foreground flex items-center justify-between">
          {installStatus === "INSTALLING" && !installedVersion ? (
            <>
              <div className="bg-muted h-4 w-24 animate-pulse rounded-sm" />
              <div className="bg-muted h-4 w-12 animate-pulse rounded-sm" />
            </>
          ) : (
            <>
              <div>
                {installedVersion
                  ? `v${installedVersion}`
                  : installStatus === "READY"
                    ? "Installed (version unknown)"
                    : "Not installed"}
              </div>
              <div>{installedSize ? formatBytes(installedSize) : null}</div>
            </>
          )}
        </div>
      </div>

      {/* Compatibility chips */}
      <div className="border-border border-b px-4 py-3">
        <div className="flex flex-wrap gap-2">
          {OS_CHIPS.map(({ key, label }) => {
            const archs = plugin.os[key];
            if (!archs) return null;
            const isHostPlatform = hostPlatform === key;
            return (
              <Badge
                key={key}
                variant={isHostPlatform ? "success" : "default"}
                className={cn("text-xs", isHostPlatform && "font-semibold")}
              >
                {label} ({archs.join(", ")})
              </Badge>
            );
          })}
        </div>
      </div>

      {/* Footer with CTA button */}
      <div className="border-border border-t px-4 py-3">
        {isDisabled ? (
          <div className="text-muted-foreground text-sm">
            Requires app {plugin.minimumHostVersion}–{plugin.maximumHostVersion} (you have{" "}
            {hostVersion})
          </div>
        ) : (
          <>
            <Button
              onClick={() => {
                if (installStatus === "NOT_INSTALLED") onInstall?.();
                else if (installStatus === "READY") {
                  if (updateAvailable) onUpdate?.();
                } else if (installStatus === "CRASHED") onRetry?.();
              }}
              disabled={
                installStatus === "INSTALLING" || (installStatus === "READY" && !updateAvailable)
              }
              className="w-full"
              variant={installStatus === "CRASHED" ? "destructive" : "default"}
            >
              {installStatus === "NOT_INSTALLED" &&
                `Install v${plugin.version} (${marketplacePluginSize})`}
              {installStatus === "INSTALLING" && "Downloading..."}
              {installStatus === "READY" &&
                (updateAvailable
                  ? `Update v${plugin.version} (${marketplacePluginSize})`
                  : "Installed")}
              {installStatus === "CRASHED" && "Retry"}
              {installStatus === "INCOMPATIBLE" && "Incompatible"}
            </Button>

            <div className="mt-2">
              <DownloadProgressIndicator
                progress={installProgress}
                isVerifying={isVerifying}
                isVisible={installStatus === "INSTALLING"}
              />
            </div>
          </>
        )}
      </div>

      <ConfirmationModal
        open={isRemoveDialogOpen}
        title={`Remove Plugin: ${plugin.name}`}
        description="This plugin will be permanently removed from your system. You can reinstall it anytime from the marketplace."
        confirmLabel="Remove"
        confirmVariant="destructive"
        isPending={isRemoving}
        onClose={() => setIsRemoveDialogOpen(false)}
        onConfirm={handleRemoveClick}
      />
    </div>
  );
};

import { InfoIcon, Tooltip, TooltipContent, TooltipTrigger, cn } from "@litelens/design-system";
import { FC } from "react";
import { useGetClusterProxy } from "./clusters/shared/hooks/data-access/useGetClusterProxy";
import { useProxyStatusEvents } from "./clusters/shared/hooks/async-events/useProxyStatusEvents";

interface Props {
  activeContext: string;
  updateInfo: { latestVersion: string; releaseURL: string } | null;
  onUpdateClick: () => void;
}

const PROXY_STATUS_LABEL: Record<string, string> = {
  starting: "Starting",
  ready: "Ready",
  degraded: "Degraded",
};

const PROXY_STATUS_DOT_CLASS: Record<string, string> = {
  starting: "bg-warning",
  ready: "bg-success",
  degraded: "bg-destructive",
};

export const AppFooter: FC<Props> = ({ activeContext, updateInfo, onUpdateClick }) => {
  const { data: clusterProxy } = useGetClusterProxy(activeContext || null);
  const { data: proxyStatus } = useProxyStatusEvents(
    clusterProxy?.setupCommand ? activeContext : null
  );

  const showProxyIndicator = !!clusterProxy?.setupCommand && !!proxyStatus;
  const proxyLabel = proxyStatus
    ? (PROXY_STATUS_LABEL[proxyStatus.State] ?? proxyStatus.State)
    : "";
  const proxyDotClass = proxyStatus
    ? (PROXY_STATUS_DOT_CLASS[proxyStatus.State] ?? "bg-muted-foreground")
    : "";

  return (
    <footer className="flex shrink-0 items-center gap-3 border-t bg-background px-3 py-2">
      {showProxyIndicator && (
        <div className="flex items-center gap-1.5 text-xs text-muted-foreground">
          <Tooltip>
            <TooltipTrigger>
              <span className={cn("inline-block size-2 rounded-full", proxyDotClass)} />
            </TooltipTrigger>
            <TooltipContent side="top" align="start">
              {proxyLabel}
              {proxyStatus?.Message ? ` — ${proxyStatus.Message}` : ""}
            </TooltipContent>
          </Tooltip>
          Proxy Server
        </div>
      )}

      {updateInfo && (
        <div className="ml-auto">
          <Tooltip>
            <TooltipTrigger
              onClick={onUpdateClick}
              className="flex items-center gap-1.5 text-xs text-primary underline-offset-2 hover:underline"
            >
              <InfoIcon className="size-3.5" />
              Update available
            </TooltipTrigger>
            <TooltipContent side="top" align="end">
              Version {updateInfo.latestVersion} is available to download
            </TooltipContent>
          </Tooltip>
        </div>
      )}
    </footer>
  );
};

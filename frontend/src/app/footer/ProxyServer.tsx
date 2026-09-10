import { Tooltip, TooltipContent, TooltipTrigger, cn } from "@litelens/design-system";
import { FC } from "react";
import { useGetClusterProxy } from "../clusters/shared/hooks/data-access/useGetClusterProxy";
import { useProxyStatusEvents } from "../clusters/shared/hooks/async-events/useProxyStatusEvents";

interface Props {
  activeContext: string;
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

export const ProxyServer: FC<Props> = ({ activeContext }) => {
  const { data: clusterProxy } = useGetClusterProxy(activeContext || null);
  const { data: proxyStatus } = useProxyStatusEvents(
    clusterProxy?.setupCommand ? activeContext : null
  );

  if (!clusterProxy?.setupCommand || !proxyStatus) return null;

  const proxyLabel = PROXY_STATUS_LABEL[proxyStatus.State] ?? proxyStatus.State;
  const proxyDotClass = PROXY_STATUS_DOT_CLASS[proxyStatus.State] ?? "bg-muted-foreground";

  return (
    <div className="flex items-center gap-1.5 text-xs text-muted-foreground">
      <Tooltip>
        <TooltipTrigger>
          <span className={cn("inline-block size-2 rounded-full", proxyDotClass)} />
        </TooltipTrigger>
        <TooltipContent side="top" align="start">
          {activeContext}: {proxyLabel}
          {proxyStatus.Message ? ` — ${proxyStatus.Message}` : ""}
        </TooltipContent>
      </Tooltip>
      Proxy Server
    </div>
  );
};

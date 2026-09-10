import { Badge, Tooltip, TooltipContent, TooltipTrigger, cn } from "@litelens/design-system";
import { FC } from "react";
import { useGetClusterProxy } from "../clusters/shared/hooks/data-access/useGetClusterProxy";
import { useProxyStatusEvents } from "../clusters/shared/hooks/async-events/useProxyStatusEvents";

interface Props {
  activeContext: string;
}

const PROXY_STATUS_LABEL: Record<string, string> = {
  idle: "Idle",
  starting: "Starting",
  ready: "Ready",
  degraded: "Degraded",
};

const PROXY_STATUS_DOT_CLASS: Record<string, string> = {
  idle: "bg-muted-foreground",
  starting: "bg-warning",
  ready: "bg-success",
  degraded: "bg-destructive",
};

const PROXY_STATUS_BADGE_VARIANT: Record<string, "ghost" | "warning" | "success" | "destructive"> =
  {
    idle: "ghost",
    starting: "warning",
    ready: "success",
    degraded: "destructive",
  };

export const ProxyServer: FC<Props> = ({ activeContext }) => {
  const { data: clusterProxy } = useGetClusterProxy(activeContext || null);
  const { data: proxyStatus } = useProxyStatusEvents(
    clusterProxy?.setupCommand ? activeContext : null
  );

  if (!clusterProxy?.setupCommand || !proxyStatus) return null;

  return (
    <div className="flex items-center gap-1.5 text-xs text-muted-foreground">
      <Tooltip>
        <TooltipTrigger>
          <span
            className={cn(
              "inline-block size-2 rounded-full",
              PROXY_STATUS_DOT_CLASS[proxyStatus.State] ?? "bg-muted-foreground"
            )}
          />
        </TooltipTrigger>
        <TooltipContent side="top" align="start">
          <div className="flex flex-col gap-1">
            <span className="flex items-center gap-1.5">
              {activeContext}{" "}
              <Badge variant={PROXY_STATUS_BADGE_VARIANT[proxyStatus.State] ?? "ghost"}>
                {PROXY_STATUS_LABEL[proxyStatus.State] ?? proxyStatus.State}
              </Badge>
            </span>
            {proxyStatus.Message && <span>{proxyStatus.Message}</span>}
          </div>
        </TooltipContent>
      </Tooltip>
      Proxy Server
    </div>
  );
};

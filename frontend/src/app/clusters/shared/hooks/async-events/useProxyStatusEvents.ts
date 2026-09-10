import { useEffect } from "react";
import { useQuery, useQueryClient } from "@tanstack/react-query";
import { GetProxyStatus } from "@wailsjs/go/app/App";
import { EventsOn } from "@wailsjs/runtime/runtime";
import { proxy } from "@wailsjs/go/models";
import { DEFAULT_QUERY_OPTIONS } from "../../../../shared/api/api";
import { QUERY_KEY_PROXY_STATUS } from "../../api/api.const";

export const useProxyStatusEvents = (contextName: string | null) => {
  const queryClient = useQueryClient();
  const queryKey = [QUERY_KEY_PROXY_STATUS, { contextName }];

  const query = useQuery<proxy.Status, Error>({
    queryKey,
    queryFn: () => GetProxyStatus(contextName!),
    ...DEFAULT_QUERY_OPTIONS,
    enabled: !!contextName,
  });

  useEffect(() => {
    if (!contextName) return;

    const applyEvent = (state: string) => (payload: { context: string; message: string }) => {
      if (payload.context !== contextName) return;
      queryClient.setQueryData<proxy.Status>(queryKey, { State: state, Message: payload.message });
    };

    const unsubscribeReady = EventsOn("SetupCommandReady", applyEvent("ready"));
    const unsubscribeDegraded = EventsOn("SetupCommandDegraded", applyEvent("degraded"));
    const unsubscribeCrashed = EventsOn("SetupCommandCrashed", applyEvent("degraded"));

    return () => {
      unsubscribeReady();
      unsubscribeDegraded();
      unsubscribeCrashed();
    };
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [contextName]);

  return query;
};

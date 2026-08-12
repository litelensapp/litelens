import { startTransition, useEffect, useState } from "react";
import { EventsOn } from "@wailsjs/runtime/runtime";
import type { ResourceQuota } from "../../api/resources";

// Data-only event hook: tracks the latest pushed resourcequotas in local state.
// Called directly from resourcequota data-access hooks (useGetResourceQuotas, useGetResourceQuotaDetail, useGetResourceQuotaYAML)
// to merge event-driven data locally without cache-wide side effects.
// Pass a namespace to subscribe to the backend's namespace-scoped channel
// ("resourcequotas:{namespace}:update") instead of the cluster-wide "resourcequotas:update" broadcast.
export function useResourceQuotasUpdateEvents(namespace = ""): ResourceQuota[] {
  const [latestResourceQuotas, setLatestResourceQuotas] = useState<ResourceQuota[]>([]);
  const [prevNamespace, setPrevNamespace] = useState(namespace);

  if (namespace !== prevNamespace) {
    setPrevNamespace(namespace);
    setLatestResourceQuotas([]);
  }

  useEffect(() => {
    const eventName = namespace ? `resourcequotas:${namespace}:update` : "resourcequotas:update";
    return EventsOn(eventName, (data: ResourceQuota[]) => {
      startTransition(() => {
        setLatestResourceQuotas(data);
      });
    });
  }, [namespace]);
  return latestResourceQuotas;
}

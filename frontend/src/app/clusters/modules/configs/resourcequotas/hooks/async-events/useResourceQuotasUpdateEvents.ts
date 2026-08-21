import { startTransition, useEffect, useState } from "react";
import { EventsOn } from "@wailsjs/runtime/runtime";
import type { ResourceQuota } from "../../api/resources";
import { mergeNamespaceScopedData } from "../../../../../shared/utils/eventMerging";

// Data-only event hook: tracks the latest pushed resourcequotas in local state.
// Called directly from resourcequota data-access hooks (useGetResourceQuotas, useGetResourceQuotaDetail, useGetResourceQuotaYAML)
// to merge event-driven data locally without cache-wide side effects.
// Pass namespaces to subscribe to the backend's namespace-scoped channels
// ("resourcequotas:{namespace}:update" for each namespace) instead of the cluster-wide "resourcequotas:update" broadcast.
export function useResourceQuotasUpdateEvents(namespaces: string[] = []): ResourceQuota[] {
  const [latestResourceQuotas, setLatestResourceQuotas] = useState<ResourceQuota[]>([]);
  const [prevNamespaces, setPrevNamespaces] = useState(namespaces);

  // When namespace selection changes, filter down accumulated state to only selected namespaces.
  if (JSON.stringify(prevNamespaces) !== JSON.stringify(namespaces)) {
    setPrevNamespaces(namespaces);
    if (namespaces.length > 0) {
      const namespacesSet = new Set(namespaces);
      setLatestResourceQuotas((prev) => prev.filter((rq) => namespacesSet.has(rq.Namespace)));
    } else {
      setLatestResourceQuotas([]);
    }
  }

  useEffect(() => {
    if (namespaces.length === 0) {
      return EventsOn("resourcequotas:update", (data: ResourceQuota[]) => {
        startTransition(() => {
          setLatestResourceQuotas(data);
        });
      });
    }

    const unsubscribers: Array<() => void> = [];
    for (const ns of namespaces) {
      const eventName = `resourcequotas:${ns}:update`;
      const unsubscriber = EventsOn(eventName, (data: ResourceQuota[]) => {
        startTransition(() => {
          setLatestResourceQuotas((prev) => mergeNamespaceScopedData(prev, data, ns));
        });
      });
      unsubscribers.push(unsubscriber);
    }

    return () => {
      unsubscribers.forEach((unsub) => unsub());
    };
  }, [namespaces]);
  return latestResourceQuotas;
}

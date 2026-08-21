import { startTransition, useEffect, useState } from "react";
import { EventsOn } from "@wailsjs/runtime/runtime";
import type { Lease } from "../../api/resources";
import { mergeNamespaceScopedData } from "../../../../../shared/utils/eventMerging";

// Data-only event hook: tracks the latest pushed leases in local state.
// Called directly from lease data-access hooks (useGetLeases, useGetLeaseByName, useGetLeaseYAML)
// to merge event-driven data locally without cache-wide side effects.
// Pass namespaces to subscribe to the backend's namespace-scoped channels
// ("leases:{namespace}:update" for each namespace) instead of the cluster-wide "leases:update" broadcast.
export function useLeasesUpdateEvents(namespaces: string[] = []): Lease[] {
  const [latestLeases, setLatestLeases] = useState<Lease[]>([]);
  const [prevNamespaces, setPrevNamespaces] = useState(namespaces);

  // When namespace selection changes, filter down accumulated state to only selected namespaces.
  if (JSON.stringify(prevNamespaces) !== JSON.stringify(namespaces)) {
    setPrevNamespaces(namespaces);
    if (namespaces.length > 0) {
      const namespacesSet = new Set(namespaces);
      setLatestLeases((prev) => prev.filter((lease) => namespacesSet.has(lease.Namespace)));
    } else {
      setLatestLeases([]);
    }
  }

  useEffect(() => {
    if (namespaces.length === 0) {
      return EventsOn("leases:update", (data: Lease[]) => {
        startTransition(() => {
          setLatestLeases(data);
        });
      });
    }

    const unsubscribers: Array<() => void> = [];
    for (const ns of namespaces) {
      const eventName = `leases:${ns}:update`;
      const unsubscriber = EventsOn(eventName, (data: Lease[]) => {
        startTransition(() => {
          setLatestLeases((prev) => mergeNamespaceScopedData(prev, data, ns));
        });
      });
      unsubscribers.push(unsubscriber);
    }

    return () => {
      unsubscribers.forEach((unsub) => unsub());
    };
  }, [namespaces]);
  return latestLeases;
}

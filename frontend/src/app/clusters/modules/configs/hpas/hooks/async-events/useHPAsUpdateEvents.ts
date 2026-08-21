import { startTransition, useEffect, useState } from "react";
import { EventsOn } from "@wailsjs/runtime/runtime";
import type { HPA } from "../../api/resources";
import { mergeNamespaceScopedData } from "../../../../../shared/utils/eventMerging";

// Data-only event hook: tracks the latest pushed hpas in local state.
// Called directly from hpa data-access hooks (useGetHPAs, useGetHPADetail, useGetHPAYAML)
// to merge event-driven data locally without cache-wide side effects.
// Pass namespaces to subscribe to the backend's namespace-scoped channels
// ("hpas:{namespace}:update" for each namespace) instead of the cluster-wide "hpas:update" broadcast.
export function useHPAsUpdateEvents(namespaces: string[] = []): HPA[] {
  const [latestHPAs, setLatestHPAs] = useState<HPA[]>([]);
  const [prevNamespaces, setPrevNamespaces] = useState(namespaces);

  // When namespace selection changes, filter down accumulated state to only selected namespaces.
  if (JSON.stringify(prevNamespaces) !== JSON.stringify(namespaces)) {
    setPrevNamespaces(namespaces);
    if (namespaces.length > 0) {
      const namespacesSet = new Set(namespaces);
      setLatestHPAs((prev) => prev.filter((hpa) => namespacesSet.has(hpa.Namespace)));
    } else {
      setLatestHPAs([]);
    }
  }

  useEffect(() => {
    if (namespaces.length === 0) {
      return EventsOn("hpas:update", (data: HPA[]) => {
        startTransition(() => {
          setLatestHPAs(data);
        });
      });
    }

    const unsubscribers: Array<() => void> = [];
    for (const ns of namespaces) {
      const eventName = `hpas:${ns}:update`;
      const unsubscriber = EventsOn(eventName, (data: HPA[]) => {
        startTransition(() => {
          setLatestHPAs((prev) => mergeNamespaceScopedData(prev, data, ns));
        });
      });
      unsubscribers.push(unsubscriber);
    }

    return () => {
      unsubscribers.forEach((unsub) => unsub());
    };
  }, [namespaces]);
  return latestHPAs;
}

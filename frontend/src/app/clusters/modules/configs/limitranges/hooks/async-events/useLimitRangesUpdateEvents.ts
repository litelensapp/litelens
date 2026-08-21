import { startTransition, useEffect, useState } from "react";
import { EventsOn } from "@wailsjs/runtime/runtime";
import type { LimitRange } from "../../api/resources";
import { mergeNamespaceScopedData } from "../../../../../shared/utils/eventMerging";

// Data-only event hook: tracks the latest pushed limitranges in local state.
// Called directly from limitrange data-access hooks (useGetLimitRanges, useGetLimitRangeDetail, useGetLimitRangeYAML)
// to merge event-driven data locally without cache-wide side effects.
// Pass namespaces to subscribe to the backend's namespace-scoped channels
// ("limitranges:{namespace}:update" for each namespace) instead of the cluster-wide "limitranges:update" broadcast.
export function useLimitRangesUpdateEvents(namespaces: string[] = []): LimitRange[] {
  const [latestLimitRanges, setLatestLimitRanges] = useState<LimitRange[]>([]);
  const [prevNamespaces, setPrevNamespaces] = useState(namespaces);

  // When namespace selection changes, filter down accumulated state to only selected namespaces.
  if (JSON.stringify(prevNamespaces) !== JSON.stringify(namespaces)) {
    setPrevNamespaces(namespaces);
    if (namespaces.length > 0) {
      const namespacesSet = new Set(namespaces);
      setLatestLimitRanges((prev) => prev.filter((lr) => namespacesSet.has(lr.Namespace)));
    } else {
      setLatestLimitRanges([]);
    }
  }

  useEffect(() => {
    if (namespaces.length === 0) {
      return EventsOn("limitranges:update", (data: LimitRange[]) => {
        startTransition(() => {
          setLatestLimitRanges(data);
        });
      });
    }

    const unsubscribers: Array<() => void> = [];
    for (const ns of namespaces) {
      const eventName = `limitranges:${ns}:update`;
      const unsubscriber = EventsOn(eventName, (data: LimitRange[]) => {
        startTransition(() => {
          setLatestLimitRanges((prev) => mergeNamespaceScopedData(prev, data, ns));
        });
      });
      unsubscribers.push(unsubscriber);
    }

    return () => {
      unsubscribers.forEach((unsub) => unsub());
    };
  }, [namespaces]);
  return latestLimitRanges;
}

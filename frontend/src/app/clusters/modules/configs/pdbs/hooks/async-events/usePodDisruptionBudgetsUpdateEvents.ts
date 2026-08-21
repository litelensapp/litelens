import { startTransition, useEffect, useState } from "react";
import { EventsOn } from "@wailsjs/runtime/runtime";
import type { PodDisruptionBudget } from "../../api/resources";
import { mergeNamespaceScopedData } from "../../../../../shared/utils/eventMerging";

// Data-only event hook: tracks the latest pushed pdbs in local state.
// Called directly from pdb data-access hooks (useGetPodDisruptionBudgets, useGetPodDisruptionBudgetDetail, useGetPDBYAML)
// to merge event-driven data locally without cache-wide side effects.
// Pass namespaces to subscribe to the backend's namespace-scoped channels
// ("pdbs:{namespace}:update" for each namespace) instead of the cluster-wide "pdbs:update" broadcast.
export function usePodDisruptionBudgetsUpdateEvents(
  namespaces: string[] = []
): PodDisruptionBudget[] {
  const [latestPodDisruptionBudgets, setLatestPodDisruptionBudgets] = useState<
    PodDisruptionBudget[]
  >([]);
  const [prevNamespaces, setPrevNamespaces] = useState(namespaces);

  // When namespace selection changes, filter down accumulated state to only selected namespaces.
  if (JSON.stringify(prevNamespaces) !== JSON.stringify(namespaces)) {
    setPrevNamespaces(namespaces);
    if (namespaces.length > 0) {
      const namespacesSet = new Set(namespaces);
      setLatestPodDisruptionBudgets((prev) =>
        prev.filter((pdb) => namespacesSet.has(pdb.Namespace))
      );
    } else {
      setLatestPodDisruptionBudgets([]);
    }
  }

  useEffect(() => {
    if (namespaces.length === 0) {
      return EventsOn("pdbs:update", (data: PodDisruptionBudget[]) => {
        startTransition(() => {
          setLatestPodDisruptionBudgets(data);
        });
      });
    }

    const unsubscribers: Array<() => void> = [];
    for (const ns of namespaces) {
      const eventName = `pdbs:${ns}:update`;
      const unsubscriber = EventsOn(eventName, (data: PodDisruptionBudget[]) => {
        startTransition(() => {
          setLatestPodDisruptionBudgets((prev) => mergeNamespaceScopedData(prev, data, ns));
        });
      });
      unsubscribers.push(unsubscriber);
    }

    return () => {
      unsubscribers.forEach((unsub) => unsub());
    };
  }, [namespaces]);
  return latestPodDisruptionBudgets;
}

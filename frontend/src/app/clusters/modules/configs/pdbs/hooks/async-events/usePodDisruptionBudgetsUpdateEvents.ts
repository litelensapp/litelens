import { startTransition, useEffect, useState } from "react";
import { EventsOn } from "@wailsjs/runtime/runtime";
import type { PodDisruptionBudget } from "../../api/resources";

// Data-only event hook: tracks the latest pushed pdbs in local state.
// Called directly from pdb data-access hooks (useGetPodDisruptionBudgets, useGetPodDisruptionBudgetDetail, useGetPDBYAML)
// to merge event-driven data locally without cache-wide side effects.
// Pass a namespace to subscribe to the backend's namespace-scoped channel
// ("pdbs:{namespace}:update") instead of the cluster-wide "pdbs:update" broadcast.
export function usePodDisruptionBudgetsUpdateEvents(namespace = ""): PodDisruptionBudget[] {
  const [latestPodDisruptionBudgets, setLatestPodDisruptionBudgets] = useState<
    PodDisruptionBudget[]
  >([]);
  const [prevNamespace, setPrevNamespace] = useState(namespace);

  if (namespace !== prevNamespace) {
    setPrevNamespace(namespace);
    setLatestPodDisruptionBudgets([]);
  }

  useEffect(() => {
    const eventName = namespace ? `pdbs:${namespace}:update` : "pdbs:update";
    return EventsOn(eventName, (data: PodDisruptionBudget[]) => {
      startTransition(() => {
        setLatestPodDisruptionBudgets(data);
      });
    });
  }, [namespace]);
  return latestPodDisruptionBudgets;
}

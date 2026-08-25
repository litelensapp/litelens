import { EventsOn } from "@wailsjs/runtime/runtime";
import { useEffect, useState, startTransition } from "react";
import type { Pod } from "../../api/resources";

// Data-only event hook: tracks the latest pushed pods in local state.
// Called directly from pod data-access hooks (useGetPods, useGetPodDetail, useGetPodYAML)
// to merge event-driven data locally without cache-wide side effects.
// The backend pre-filters "pods:update" by the currently active namespace
// selection (see App.SetActiveNamespaces / emitPodsWithMetrics), so this hook
// no longer needs to know about namespaces at all.
export function usePodsUpdateEvents(): Pod[] {
  const [latestPods, setLatestPods] = useState<Pod[]>([]);

  useEffect(() => {
    return EventsOn("pods:update", (data: Pod[]) => {
      startTransition(() => {
        setLatestPods(data);
      });
    });
  }, []);

  return latestPods;
}

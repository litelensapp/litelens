import { EventsOn } from "@wailsjs/runtime/runtime";
import { useEffect, useState, startTransition } from "react";
import type { Pod } from "../../api/resources";

// Data-only event hook: tracks the latest pushed pods in local state.
// Called directly from pod data-access hooks (useGetPods, useGetPodDetail, useGetPodYAML)
// to merge event-driven data locally without cache-wide side effects.
// Pass a namespace to subscribe to the backend's namespace-scoped channel
// ("pods:{namespace}:update") instead of the cluster-wide "pods:update" broadcast.
export function usePodsUpdateEvents(namespace = ""): Pod[] {
  const [latestPods, setLatestPods] = useState<Pod[]>([]);
  const [prevNamespace, setPrevNamespace] = useState(namespace);

  // Reset stale data from the previous namespace's channel before re-subscribing.
  if (namespace !== prevNamespace) {
    setPrevNamespace(namespace);
    setLatestPods([]);
  }

  useEffect(() => {
    const eventName = namespace ? `pods:${namespace}:update` : "pods:update";
    return EventsOn(eventName, (data: Pod[]) => {
      startTransition(() => {
        setLatestPods(data);
      });
    });
  }, [namespace]);

  return latestPods;
}

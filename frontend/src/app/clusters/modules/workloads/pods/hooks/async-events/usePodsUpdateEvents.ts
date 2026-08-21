import { EventsOn } from "@wailsjs/runtime/runtime";
import { useEffect, useState, startTransition } from "react";
import type { Pod } from "../../api/resources";
import { mergeNamespaceScopedData } from "../../../../../shared/utils/eventMerging";

// Data-only event hook: tracks the latest pushed pods in local state.
// Called directly from pod data-access hooks (useGetPods, useGetPodDetail, useGetPodYAML)
// to merge event-driven data locally without cache-wide side effects.
// Pass namespaces to subscribe to the backend's namespace-scoped channels
// ("pods:{namespace}:update" for each namespace) instead of the cluster-wide "pods:update" broadcast.
export function usePodsUpdateEvents(namespaces: string[] = []): Pod[] {
  const [latestPods, setLatestPods] = useState<Pod[]>([]);
  const [prevNamespaces, setPrevNamespaces] = useState(namespaces);

  // When namespace selection changes, filter down accumulated state to only selected namespaces.
  if (JSON.stringify(prevNamespaces) !== JSON.stringify(namespaces)) {
    setPrevNamespaces(namespaces);
    if (namespaces.length > 0) {
      const namespacesSet = new Set(namespaces);
      setLatestPods((prev) => prev.filter((pod) => namespacesSet.has(pod.Namespace)));
    } else {
      setLatestPods([]);
    }
  }

  useEffect(() => {
    if (namespaces.length === 0) {
      return EventsOn("pods:update", (data: Pod[]) => {
        startTransition(() => {
          setLatestPods(data);
        });
      });
    }

    const unsubscribers: Array<() => void> = [];
    for (const ns of namespaces) {
      const eventName = `pods:${ns}:update`;
      const unsubscriber = EventsOn(eventName, (data: Pod[]) => {
        startTransition(() => {
          setLatestPods((prev) => mergeNamespaceScopedData(prev, data, ns));
        });
      });
      unsubscribers.push(unsubscriber);
    }

    return () => {
      unsubscribers.forEach((unsub) => unsub());
    };
  }, [namespaces]);

  return latestPods;
}

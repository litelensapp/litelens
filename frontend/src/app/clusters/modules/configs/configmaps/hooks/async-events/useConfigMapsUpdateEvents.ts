import { startTransition, useEffect, useState } from "react";
import { EventsOn } from "@wailsjs/runtime/runtime";
import type { ConfigMap } from "../../api/resources";
import { mergeNamespaceScopedData } from "../../../../../shared/utils/eventMerging";

// Data-only event hook: tracks the latest pushed configmaps in local state.
// Called directly from configmap data-access hooks (useGetConfigMaps, useGetConfigMapDetail, useGetConfigMapYAML)
// to merge event-driven data locally without cache-wide side effects.
// Pass namespaces to subscribe to the backend's namespace-scoped channels
// ("configmaps:{namespace}:update" for each namespace) instead of the cluster-wide "configmaps:update" broadcast.
export function useConfigMapsUpdateEvents(namespaces: string[] = []): ConfigMap[] {
  const [latestConfigMaps, setLatestConfigMaps] = useState<ConfigMap[]>([]);
  const [prevNamespaces, setPrevNamespaces] = useState(namespaces);

  // When namespace selection changes, filter down accumulated state to only selected namespaces.
  if (JSON.stringify(prevNamespaces) !== JSON.stringify(namespaces)) {
    setPrevNamespaces(namespaces);
    if (namespaces.length > 0) {
      const namespacesSet = new Set(namespaces);
      setLatestConfigMaps((prev) => prev.filter((cm) => namespacesSet.has(cm.Namespace)));
    } else {
      setLatestConfigMaps([]);
    }
  }

  useEffect(() => {
    if (namespaces.length === 0) {
      return EventsOn("configmaps:update", (data: ConfigMap[]) => {
        startTransition(() => {
          setLatestConfigMaps(data);
        });
      });
    }

    const unsubscribers: Array<() => void> = [];
    for (const ns of namespaces) {
      const eventName = `configmaps:${ns}:update`;
      const unsubscriber = EventsOn(eventName, (data: ConfigMap[]) => {
        startTransition(() => {
          setLatestConfigMaps((prev) => mergeNamespaceScopedData(prev, data, ns));
        });
      });
      unsubscribers.push(unsubscriber);
    }

    return () => {
      unsubscribers.forEach((unsub) => unsub());
    };
  }, [namespaces]);
  return latestConfigMaps;
}

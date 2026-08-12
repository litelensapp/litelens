import { startTransition, useEffect, useState } from "react";
import { EventsOn } from "@wailsjs/runtime/runtime";
import type { ConfigMap } from "../../api/resources";

// Data-only event hook: tracks the latest pushed configmaps in local state.
// Called directly from configmap data-access hooks (useGetConfigMaps, useGetConfigMapDetail, useGetConfigMapYAML)
// to merge event-driven data locally without cache-wide side effects.
// Pass a namespace to subscribe to the backend's namespace-scoped channel
// ("configmaps:{namespace}:update") instead of the cluster-wide "configmaps:update" broadcast.
export function useConfigMapsUpdateEvents(namespace = ""): ConfigMap[] {
  const [latestConfigMaps, setLatestConfigMaps] = useState<ConfigMap[]>([]);
  const [prevNamespace, setPrevNamespace] = useState(namespace);

  // Reset stale data from the previous namespace's channel before re-subscribing.
  if (namespace !== prevNamespace) {
    setPrevNamespace(namespace);
    setLatestConfigMaps([]);
  }

  useEffect(() => {
    const eventName = namespace ? `configmaps:${namespace}:update` : "configmaps:update";
    return EventsOn(eventName, (data: ConfigMap[]) => {
      startTransition(() => {
        setLatestConfigMaps(data);
      });
    });
  }, [namespace]);
  return latestConfigMaps;
}

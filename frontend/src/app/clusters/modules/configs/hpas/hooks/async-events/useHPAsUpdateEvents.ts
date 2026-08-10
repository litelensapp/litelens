import { useEffect, useState } from "react";
import { EventsOn } from "@wailsjs/runtime/runtime";
import type { HPA } from "../../api/resources";

// Data-only event hook: tracks the latest pushed hpas in local state.
// Called directly from hpa data-access hooks (useGetHPAs, useGetHPADetail, useGetHPAYAML)
// to merge event-driven data locally without cache-wide side effects.
// Pass a namespace to subscribe to the backend's namespace-scoped channel
// ("hpas:{namespace}:update") instead of the cluster-wide "hpas:update" broadcast.
export function useHPAsUpdateEvents(namespace = ""): HPA[] {
  const [latestHPAs, setLatestHPAs] = useState<HPA[]>([]);
  const [prevNamespace, setPrevNamespace] = useState(namespace);

  if (namespace !== prevNamespace) {
    setPrevNamespace(namespace);
    setLatestHPAs([]);
  }

  useEffect(() => {
    const eventName = namespace ? `hpas:${namespace}:update` : "hpas:update";
    return EventsOn(eventName, (data: HPA[]) => setLatestHPAs(data));
  }, [namespace]);
  return latestHPAs;
}

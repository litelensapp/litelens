import { useEffect, useState } from "react";
import { EventsOn } from "@wailsjs/runtime/runtime";
import type { LimitRange } from "../../api/resources";

// Data-only event hook: tracks the latest pushed limitranges in local state.
// Called directly from limitrange data-access hooks (useGetLimitRanges, useGetLimitRangeDetail, useGetLimitRangeYAML)
// to merge event-driven data locally without cache-wide side effects.
// Pass a namespace to subscribe to the backend's namespace-scoped channel
// ("limitranges:{namespace}:update") instead of the cluster-wide "limitranges:update" broadcast.
export function useLimitRangesUpdateEvents(namespace = ""): LimitRange[] {
  const [latestLimitRanges, setLatestLimitRanges] = useState<LimitRange[]>([]);
  const [prevNamespace, setPrevNamespace] = useState(namespace);

  if (namespace !== prevNamespace) {
    setPrevNamespace(namespace);
    setLatestLimitRanges([]);
  }

  useEffect(() => {
    const eventName = namespace ? `limitranges:${namespace}:update` : "limitranges:update";
    return EventsOn(eventName, (data: LimitRange[]) => setLatestLimitRanges(data));
  }, [namespace]);
  return latestLimitRanges;
}

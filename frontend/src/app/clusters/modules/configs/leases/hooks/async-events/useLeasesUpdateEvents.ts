import { useEffect, useState } from "react";
import { EventsOn } from "@wailsjs/runtime/runtime";
import type { Lease } from "../../api/resources";

// Data-only event hook: tracks the latest pushed leases in local state.
// Called directly from lease data-access hooks (useGetLeases, useGetLeaseByName, useGetLeaseYAML)
// to merge event-driven data locally without cache-wide side effects.
// Pass a namespace to subscribe to the backend's namespace-scoped channel
// ("leases:{namespace}:update") instead of the cluster-wide "leases:update" broadcast.
export function useLeasesUpdateEvents(namespace = ""): Lease[] {
  const [latestLeases, setLatestLeases] = useState<Lease[]>([]);
  const [prevNamespace, setPrevNamespace] = useState(namespace);

  if (namespace !== prevNamespace) {
    setPrevNamespace(namespace);
    setLatestLeases([]);
  }

  useEffect(() => {
    const eventName = namespace ? `leases:${namespace}:update` : "leases:update";
    return EventsOn(eventName, (data: Lease[]) => setLatestLeases(data));
  }, [namespace]);
  return latestLeases;
}

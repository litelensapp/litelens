import { useEffect, useState } from "react";
import { EventsOn } from "@wailsjs/runtime/runtime";
import type { PersistentVolumeClaim } from "../../api/resources";

// Data-only event hook: tracks the latest pushed pvcs in local state.
// Called directly from pvc data-access hooks (useGetPersistentVolumeClaims, useGetPersistentVolumeClaimDetail, useGetPersistentVolumeClaimYAML)
// to merge event-driven data locally without cache-wide side effects.
// Pass a namespace to subscribe to the backend's namespace-scoped channel
// ("pvcs:{namespace}:update") instead of the cluster-wide "persistentvolumeclaims:update" broadcast.
export function usePersistentVolumeClaimsUpdateEvents(namespace = ""): PersistentVolumeClaim[] {
  const [latestPVCs, setLatestPVCs] = useState<PersistentVolumeClaim[]>([]);
  const [prevNamespace, setPrevNamespace] = useState(namespace);

  // Reset stale data from the previous namespace's channel before re-subscribing.
  if (namespace !== prevNamespace) {
    setPrevNamespace(namespace);
    setLatestPVCs([]);
  }

  useEffect(() => {
    const eventName = namespace ? `pvcs:${namespace}:update` : "persistentvolumeclaims:update";
    return EventsOn(eventName, (data: PersistentVolumeClaim[]) => setLatestPVCs(data));
  }, [namespace]);
  return latestPVCs;
}

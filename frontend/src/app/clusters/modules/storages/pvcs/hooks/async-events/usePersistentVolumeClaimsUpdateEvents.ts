import { startTransition, useEffect, useState } from "react";
import { EventsOn } from "@wailsjs/runtime/runtime";
import type { PersistentVolumeClaim } from "../../api/resources";
import { mergeNamespaceScopedData } from "../../../../../shared/utils/eventMerging";

// Data-only event hook: tracks the latest pushed pvcs in local state.
// Called directly from pvc data-access hooks (useGetPersistentVolumeClaims, useGetPersistentVolumeClaimDetail, useGetPersistentVolumeClaimYAML)
// to merge event-driven data locally without cache-wide side effects.
// Pass namespaces to subscribe to the backend's namespace-scoped channels
// ("pvcs:{namespace}:update" for each namespace) instead of the cluster-wide "persistentvolumeclaims:update" broadcast.
export function usePersistentVolumeClaimsUpdateEvents(
  namespaces: string[] = []
): PersistentVolumeClaim[] {
  const [latestPVCs, setLatestPVCs] = useState<PersistentVolumeClaim[]>([]);
  const [prevNamespaces, setPrevNamespaces] = useState(namespaces);

  // When namespace selection changes, filter down accumulated state to only selected namespaces.
  if (JSON.stringify(prevNamespaces) !== JSON.stringify(namespaces)) {
    setPrevNamespaces(namespaces);
    if (namespaces.length > 0) {
      const namespacesSet = new Set(namespaces);
      setLatestPVCs((prev) => prev.filter((pvc) => namespacesSet.has(pvc.Namespace)));
    } else {
      setLatestPVCs([]);
    }
  }

  useEffect(() => {
    if (namespaces.length === 0) {
      return EventsOn("persistentvolumeclaims:update", (data: PersistentVolumeClaim[]) => {
        startTransition(() => {
          setLatestPVCs(data);
        });
      });
    }

    const unsubscribers: Array<() => void> = [];
    for (const ns of namespaces) {
      const eventName = `pvcs:${ns}:update`;
      const unsubscriber = EventsOn(eventName, (data: PersistentVolumeClaim[]) => {
        startTransition(() => {
          setLatestPVCs((prev) => mergeNamespaceScopedData(prev, data, ns));
        });
      });
      unsubscribers.push(unsubscriber);
    }

    return () => {
      unsubscribers.forEach((unsub) => unsub());
    };
  }, [namespaces]);
  return latestPVCs;
}

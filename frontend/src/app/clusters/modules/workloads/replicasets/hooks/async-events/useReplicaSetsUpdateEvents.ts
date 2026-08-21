import { useEffect, useState, startTransition } from "react";
import { EventsOn } from "@wailsjs/runtime/runtime";
import type { ReplicaSet } from "../../api/resources";
import { mergeNamespaceScopedData } from "../../../../../shared/utils/eventMerging";

export function useReplicaSetsUpdateEvents(namespaces: string[] = []): ReplicaSet[] {
  const [latestReplicaSets, setlatestReplicaSets] = useState<ReplicaSet[]>([]);
  const [prevNamespaces, setPrevNamespaces] = useState(namespaces);

  // When namespace selection changes, filter down accumulated state to only selected namespaces.
  if (JSON.stringify(prevNamespaces) !== JSON.stringify(namespaces)) {
    setPrevNamespaces(namespaces);
    if (namespaces.length > 0) {
      const namespacesSet = new Set(namespaces);
      setlatestReplicaSets((prev) => prev.filter((item) => namespacesSet.has(item.Namespace)));
    } else {
      setlatestReplicaSets([]);
    }
  }

  useEffect(() => {
    if (namespaces.length === 0) {
      return EventsOn("replicasets:update", (data: ReplicaSet[]) => {
        startTransition(() => {
          setlatestReplicaSets(data);
        });
      });
    }

    const unsubscribers: Array<() => void> = [];
    for (const ns of namespaces) {
      const eventName = `replicasets:${ns}:update`;
      const unsubscriber = EventsOn(eventName, (data: ReplicaSet[]) => {
        startTransition(() => {
          setlatestReplicaSets((prev) => mergeNamespaceScopedData(prev, data, ns));
        });
      });
      unsubscribers.push(unsubscriber);
    }

    return () => {
      unsubscribers.forEach((unsub) => unsub());
    };
  }, [namespaces]);
  return latestReplicaSets;
}

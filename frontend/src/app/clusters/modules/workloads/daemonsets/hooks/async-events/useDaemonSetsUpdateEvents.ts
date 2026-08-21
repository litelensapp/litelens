import { useEffect, useState, startTransition } from "react";
import { EventsOn } from "@wailsjs/runtime/runtime";
import type { DaemonSet } from "../../api/resources";
import { mergeNamespaceScopedData } from "../../../../../shared/utils/eventMerging";

export function useDaemonSetsUpdateEvents(namespaces: string[] = []): DaemonSet[] {
  const [latestDaemonSets, setlatestDaemonSets] = useState<DaemonSet[]>([]);
  const [prevNamespaces, setPrevNamespaces] = useState(namespaces);

  // When namespace selection changes, filter down accumulated state to only selected namespaces.
  if (JSON.stringify(prevNamespaces) !== JSON.stringify(namespaces)) {
    setPrevNamespaces(namespaces);
    if (namespaces.length > 0) {
      const namespacesSet = new Set(namespaces);
      setlatestDaemonSets((prev) => prev.filter((item) => namespacesSet.has(item.Namespace)));
    } else {
      setlatestDaemonSets([]);
    }
  }

  useEffect(() => {
    if (namespaces.length === 0) {
      return EventsOn("daemonsets:update", (data: DaemonSet[]) => {
        startTransition(() => {
          setlatestDaemonSets(data);
        });
      });
    }

    const unsubscribers: Array<() => void> = [];
    for (const ns of namespaces) {
      const eventName = `daemonsets:${ns}:update`;
      const unsubscriber = EventsOn(eventName, (data: DaemonSet[]) => {
        startTransition(() => {
          setlatestDaemonSets((prev) => mergeNamespaceScopedData(prev, data, ns));
        });
      });
      unsubscribers.push(unsubscriber);
    }

    return () => {
      unsubscribers.forEach((unsub) => unsub());
    };
  }, [namespaces]);
  return latestDaemonSets;
}

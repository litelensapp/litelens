import { useEffect, useState, startTransition } from "react";
import { EventsOn } from "@wailsjs/runtime/runtime";
import type { StatefulSet } from "../../api/resources";
import { mergeNamespaceScopedData } from "../../../../../shared/utils/eventMerging";

export function useStatefulSetsUpdateEvents(namespaces: string[] = []): StatefulSet[] {
  const [latestStatefulSets, setlatestStatefulSets] = useState<StatefulSet[]>([]);
  const [prevNamespaces, setPrevNamespaces] = useState(namespaces);

  // When namespace selection changes, filter down accumulated state to only selected namespaces.
  if (JSON.stringify(prevNamespaces) !== JSON.stringify(namespaces)) {
    setPrevNamespaces(namespaces);
    if (namespaces.length > 0) {
      const namespacesSet = new Set(namespaces);
      setlatestStatefulSets((prev) => prev.filter((item) => namespacesSet.has(item.Namespace)));
    } else {
      setlatestStatefulSets([]);
    }
  }

  useEffect(() => {
    if (namespaces.length === 0) {
      return EventsOn("statefulsets:update", (data: StatefulSet[]) => {
        startTransition(() => {
          setlatestStatefulSets(data);
        });
      });
    }

    const unsubscribers: Array<() => void> = [];
    for (const ns of namespaces) {
      const eventName = `statefulsets:${ns}:update`;
      const unsubscriber = EventsOn(eventName, (data: StatefulSet[]) => {
        startTransition(() => {
          setlatestStatefulSets((prev) => mergeNamespaceScopedData(prev, data, ns));
        });
      });
      unsubscribers.push(unsubscriber);
    }

    return () => {
      unsubscribers.forEach((unsub) => unsub());
    };
  }, [namespaces]);
  return latestStatefulSets;
}

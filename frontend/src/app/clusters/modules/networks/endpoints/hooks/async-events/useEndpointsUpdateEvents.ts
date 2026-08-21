import { useEffect, useState, startTransition } from "react";
import { EventsOn } from "@wailsjs/runtime/runtime";
import type { Endpoint } from "../../api/resources";
import { mergeNamespaceScopedData } from "../../../../../shared/utils/eventMerging";

export function useEndpointsUpdateEvents(namespaces: string[] = []): Endpoint[] {
  const [latestEndpoints, setlatestEndpoints] = useState<Endpoint[]>([]);
  const [prevNamespaces, setPrevNamespaces] = useState(namespaces);

  // When namespace selection changes, filter down accumulated state to only selected namespaces.
  if (JSON.stringify(prevNamespaces) !== JSON.stringify(namespaces)) {
    setPrevNamespaces(namespaces);
    if (namespaces.length > 0) {
      const namespacesSet = new Set(namespaces);
      setlatestEndpoints((prev) => prev.filter((item) => namespacesSet.has(item.Namespace)));
    } else {
      setlatestEndpoints([]);
    }
  }

  useEffect(() => {
    if (namespaces.length === 0) {
      return EventsOn("endpoints:update", (data: Endpoint[]) => {
        startTransition(() => {
          setlatestEndpoints(data);
        });
      });
    }

    const unsubscribers: Array<() => void> = [];
    for (const ns of namespaces) {
      const eventName = `endpoints:${ns}:update`;
      const unsubscriber = EventsOn(eventName, (data: Endpoint[]) => {
        startTransition(() => {
          setlatestEndpoints((prev) => mergeNamespaceScopedData(prev, data, ns));
        });
      });
      unsubscribers.push(unsubscriber);
    }

    return () => {
      unsubscribers.forEach((unsub) => unsub());
    };
  }, [namespaces]);
  return latestEndpoints;
}

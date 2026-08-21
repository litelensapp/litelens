import { useEffect, useState, startTransition } from "react";
import { EventsOn } from "@wailsjs/runtime/runtime";
import type { EndpointSlice } from "../../api/resources";
import { mergeNamespaceScopedData } from "../../../../../shared/utils/eventMerging";

export function useEndpointSlicesUpdateEvents(namespaces: string[] = []): EndpointSlice[] {
  const [latestEndpointSlices, setlatestEndpointSlices] = useState<EndpointSlice[]>([]);
  const [prevNamespaces, setPrevNamespaces] = useState(namespaces);

  // When namespace selection changes, filter down accumulated state to only selected namespaces.
  if (JSON.stringify(prevNamespaces) !== JSON.stringify(namespaces)) {
    setPrevNamespaces(namespaces);
    if (namespaces.length > 0) {
      const namespacesSet = new Set(namespaces);
      setlatestEndpointSlices((prev) => prev.filter((item) => namespacesSet.has(item.Namespace)));
    } else {
      setlatestEndpointSlices([]);
    }
  }

  useEffect(() => {
    if (namespaces.length === 0) {
      return EventsOn("endpointslices:update", (data: EndpointSlice[]) => {
        startTransition(() => {
          setlatestEndpointSlices(data);
        });
      });
    }

    const unsubscribers: Array<() => void> = [];
    for (const ns of namespaces) {
      const eventName = `endpointslices:${ns}:update`;
      const unsubscriber = EventsOn(eventName, (data: EndpointSlice[]) => {
        startTransition(() => {
          setlatestEndpointSlices((prev) => mergeNamespaceScopedData(prev, data, ns));
        });
      });
      unsubscribers.push(unsubscriber);
    }

    return () => {
      unsubscribers.forEach((unsub) => unsub());
    };
  }, [namespaces]);
  return latestEndpointSlices;
}

import { useEffect, useState, startTransition } from "react";
import { EventsOn } from "@wailsjs/runtime/runtime";
import type { Service } from "../../api/resources";
import { mergeNamespaceScopedData } from "../../../../../shared/utils/eventMerging";

export function useServicesUpdateEvents(namespaces: string[] = []): Service[] {
  const [latestServices, setlatestServices] = useState<Service[]>([]);
  const [prevNamespaces, setPrevNamespaces] = useState(namespaces);

  // When namespace selection changes, filter down accumulated state to only selected namespaces.
  if (JSON.stringify(prevNamespaces) !== JSON.stringify(namespaces)) {
    setPrevNamespaces(namespaces);
    if (namespaces.length > 0) {
      const namespacesSet = new Set(namespaces);
      setlatestServices((prev) => prev.filter((item) => namespacesSet.has(item.Namespace)));
    } else {
      setlatestServices([]);
    }
  }

  useEffect(() => {
    if (namespaces.length === 0) {
      return EventsOn("services:update", (data: Service[]) => {
        startTransition(() => {
          setlatestServices(data);
        });
      });
    }

    const unsubscribers: Array<() => void> = [];
    for (const ns of namespaces) {
      const eventName = `services:${ns}:update`;
      const unsubscriber = EventsOn(eventName, (data: Service[]) => {
        startTransition(() => {
          setlatestServices((prev) => mergeNamespaceScopedData(prev, data, ns));
        });
      });
      unsubscribers.push(unsubscriber);
    }

    return () => {
      unsubscribers.forEach((unsub) => unsub());
    };
  }, [namespaces]);
  return latestServices;
}

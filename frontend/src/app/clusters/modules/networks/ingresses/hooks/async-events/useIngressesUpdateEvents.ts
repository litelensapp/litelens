import { useEffect, useState, startTransition } from "react";
import { EventsOn } from "@wailsjs/runtime/runtime";
import type { Ingress } from "../../api/resources";
import { mergeNamespaceScopedData } from "../../../../../shared/utils/eventMerging";

export function useIngressesUpdateEvents(namespaces: string[] = []): Ingress[] {
  const [latestIngresses, setlatestIngresses] = useState<Ingress[]>([]);
  const [prevNamespaces, setPrevNamespaces] = useState(namespaces);

  // When namespace selection changes, filter down accumulated state to only selected namespaces.
  if (JSON.stringify(prevNamespaces) !== JSON.stringify(namespaces)) {
    setPrevNamespaces(namespaces);
    if (namespaces.length > 0) {
      const namespacesSet = new Set(namespaces);
      setlatestIngresses((prev) => prev.filter((item) => namespacesSet.has(item.Namespace)));
    } else {
      setlatestIngresses([]);
    }
  }

  useEffect(() => {
    if (namespaces.length === 0) {
      return EventsOn("ingresses:update", (data: Ingress[]) => {
        startTransition(() => {
          setlatestIngresses(data);
        });
      });
    }

    const unsubscribers: Array<() => void> = [];
    for (const ns of namespaces) {
      const eventName = `ingresses:${ns}:update`;
      const unsubscriber = EventsOn(eventName, (data: Ingress[]) => {
        startTransition(() => {
          setlatestIngresses((prev) => mergeNamespaceScopedData(prev, data, ns));
        });
      });
      unsubscribers.push(unsubscriber);
    }

    return () => {
      unsubscribers.forEach((unsub) => unsub());
    };
  }, [namespaces]);
  return latestIngresses;
}

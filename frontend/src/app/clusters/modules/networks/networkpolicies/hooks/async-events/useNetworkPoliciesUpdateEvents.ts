import { useEffect, useState, startTransition } from "react";
import { EventsOn } from "@wailsjs/runtime/runtime";
import type { NetworkPolicy } from "../../api/resources";
import { mergeNamespaceScopedData } from "../../../../../shared/utils/eventMerging";

export function useNetworkPoliciesUpdateEvents(namespaces: string[] = []): NetworkPolicy[] {
  const [latestNetworkPolicies, setlatestNetworkPolicies] = useState<NetworkPolicy[]>([]);
  const [prevNamespaces, setPrevNamespaces] = useState(namespaces);

  // When namespace selection changes, filter down accumulated state to only selected namespaces.
  if (JSON.stringify(prevNamespaces) !== JSON.stringify(namespaces)) {
    setPrevNamespaces(namespaces);
    if (namespaces.length > 0) {
      const namespacesSet = new Set(namespaces);
      setlatestNetworkPolicies((prev) => prev.filter((item) => namespacesSet.has(item.Namespace)));
    } else {
      setlatestNetworkPolicies([]);
    }
  }

  useEffect(() => {
    if (namespaces.length === 0) {
      return EventsOn("networkpolicies:update", (data: NetworkPolicy[]) => {
        startTransition(() => {
          setlatestNetworkPolicies(data);
        });
      });
    }

    const unsubscribers: Array<() => void> = [];
    for (const ns of namespaces) {
      const eventName = `networkpolicies:${ns}:update`;
      const unsubscriber = EventsOn(eventName, (data: NetworkPolicy[]) => {
        startTransition(() => {
          setlatestNetworkPolicies((prev) => mergeNamespaceScopedData(prev, data, ns));
        });
      });
      unsubscribers.push(unsubscriber);
    }

    return () => {
      unsubscribers.forEach((unsub) => unsub());
    };
  }, [namespaces]);
  return latestNetworkPolicies;
}

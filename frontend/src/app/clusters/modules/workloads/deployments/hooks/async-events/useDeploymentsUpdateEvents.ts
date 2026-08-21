import { useEffect, useState, startTransition } from "react";
import { EventsOn } from "@wailsjs/runtime/runtime";
import type { Deployment } from "../../api/resources";
import { mergeNamespaceScopedData } from "../../../../../shared/utils/eventMerging";

export function useDeploymentsUpdateEvents(namespaces: string[] = []): Deployment[] {
  const [latestDeployments, setLatestDeployments] = useState<Deployment[]>([]);
  const [prevNamespaces, setPrevNamespaces] = useState(namespaces);

  // When namespace selection changes, filter down accumulated state to only selected namespaces.
  if (JSON.stringify(prevNamespaces) !== JSON.stringify(namespaces)) {
    setPrevNamespaces(namespaces);
    if (namespaces.length > 0) {
      const namespacesSet = new Set(namespaces);
      setLatestDeployments((prev) => prev.filter((d) => namespacesSet.has(d.Namespace)));
    } else {
      setLatestDeployments([]);
    }
  }

  useEffect(() => {
    if (namespaces.length === 0) {
      return EventsOn("deployments:update", (data: Deployment[]) => {
        startTransition(() => {
          setLatestDeployments(data);
        });
      });
    }

    const unsubscribers: Array<() => void> = [];
    for (const ns of namespaces) {
      const eventName = `deployments:${ns}:update`;
      const unsubscriber = EventsOn(eventName, (data: Deployment[]) => {
        startTransition(() => {
          setLatestDeployments((prev) => mergeNamespaceScopedData(prev, data, ns));
        });
      });
      unsubscribers.push(unsubscriber);
    }

    return () => {
      unsubscribers.forEach((unsub) => unsub());
    };
  }, [namespaces]);
  return latestDeployments;
}

import { startTransition, useEffect, useState } from "react";
import { EventsOn } from "@wailsjs/runtime/runtime";
import type { Secret } from "../../api/resources";
import { mergeNamespaceScopedData } from "../../../../../shared/utils/eventMerging";

// Data-only event hook: tracks the latest pushed secrets in local state.
// Called directly from secret data-access hooks (useGetSecrets, useGetSecretDetail, useGetSecretYAML)
// to merge event-driven data locally without cache-wide side effects.
// Pass namespaces to subscribe to the backend's namespace-scoped channels
// ("secrets:{namespace}:update" for each namespace) instead of the cluster-wide "secrets:update" broadcast.
export function useSecretsUpdateEvents(namespaces: string[] = []): Secret[] {
  const [latestSecrets, setLatestSecrets] = useState<Secret[]>([]);
  const [prevNamespaces, setPrevNamespaces] = useState(namespaces);

  // When namespace selection changes, filter down accumulated state to only selected namespaces.
  if (JSON.stringify(prevNamespaces) !== JSON.stringify(namespaces)) {
    setPrevNamespaces(namespaces);
    if (namespaces.length > 0) {
      const namespacesSet = new Set(namespaces);
      setLatestSecrets((prev) => prev.filter((secret) => namespacesSet.has(secret.Namespace)));
    } else {
      setLatestSecrets([]);
    }
  }

  useEffect(() => {
    if (namespaces.length === 0) {
      return EventsOn("secrets:update", (data: Secret[]) => {
        startTransition(() => {
          setLatestSecrets(data);
        });
      });
    }

    const unsubscribers: Array<() => void> = [];
    for (const ns of namespaces) {
      const eventName = `secrets:${ns}:update`;
      const unsubscriber = EventsOn(eventName, (data: Secret[]) => {
        startTransition(() => {
          setLatestSecrets((prev) => mergeNamespaceScopedData(prev, data, ns));
        });
      });
      unsubscribers.push(unsubscriber);
    }

    return () => {
      unsubscribers.forEach((unsub) => unsub());
    };
  }, [namespaces]);
  return latestSecrets;
}

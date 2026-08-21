import { startTransition, useEffect, useState } from "react";
import { EventsOn } from "@wailsjs/runtime/runtime";
import type { ServiceAccount } from "../../api/resources";
import { mergeNamespaceScopedData } from "../../../../../shared/utils/eventMerging";

// Data-only event hook: tracks the latest pushed serviceaccounts in local state.
// Called directly from serviceaccount data-access hooks (useGetServiceAccounts, useGetServiceAccountDetail, useGetServiceAccountYAML)
// to merge event-driven data locally without cache-wide side effects.
// Pass namespaces to subscribe to the backend's namespace-scoped channels
// ("serviceaccounts:{namespace}:update" for each namespace) instead of the cluster-wide "serviceaccounts:update" broadcast.
export function useServiceAccountsUpdateEvents(namespaces: string[] = []): ServiceAccount[] {
  const [latestServiceAccounts, setLatestServiceAccounts] = useState<ServiceAccount[]>([]);
  const [prevNamespaces, setPrevNamespaces] = useState(namespaces);

  // When namespace selection changes, filter down accumulated state to only selected namespaces.
  if (JSON.stringify(prevNamespaces) !== JSON.stringify(namespaces)) {
    setPrevNamespaces(namespaces);
    if (namespaces.length > 0) {
      const namespacesSet = new Set(namespaces);
      setLatestServiceAccounts((prev) => prev.filter((sa) => namespacesSet.has(sa.Namespace)));
    } else {
      setLatestServiceAccounts([]);
    }
  }

  useEffect(() => {
    if (namespaces.length === 0) {
      return EventsOn("serviceaccounts:update", (data: ServiceAccount[]) => {
        startTransition(() => {
          setLatestServiceAccounts(data);
        });
      });
    }

    const unsubscribers: Array<() => void> = [];
    for (const ns of namespaces) {
      const eventName = `serviceaccounts:${ns}:update`;
      const unsubscriber = EventsOn(eventName, (data: ServiceAccount[]) => {
        startTransition(() => {
          setLatestServiceAccounts((prev) => mergeNamespaceScopedData(prev, data, ns));
        });
      });
      unsubscribers.push(unsubscriber);
    }

    return () => {
      unsubscribers.forEach((unsub) => unsub());
    };
  }, [namespaces]);
  return latestServiceAccounts;
}

import { useEffect, useState } from "react";
import { EventsOn } from "@wailsjs/runtime/runtime";
import type { ServiceAccount } from "../../api/resources";

// Data-only event hook: tracks the latest pushed serviceaccounts in local state.
// Called directly from serviceaccount data-access hooks (useGetServiceAccounts, useGetServiceAccountDetail, useGetServiceAccountYAML)
// to merge event-driven data locally without cache-wide side effects.
// Pass a namespace to subscribe to the backend's namespace-scoped channel
// ("serviceaccounts:{namespace}:update") instead of the cluster-wide "serviceaccounts:update" broadcast.
export function useServiceAccountsUpdateEvents(namespace = ""): ServiceAccount[] {
  const [latestServiceAccounts, setLatestServiceAccounts] = useState<ServiceAccount[]>([]);
  const [prevNamespace, setPrevNamespace] = useState(namespace);

  // Reset stale data from the previous namespace's channel before re-subscribing.
  if (namespace !== prevNamespace) {
    setPrevNamespace(namespace);
    setLatestServiceAccounts([]);
  }

  useEffect(() => {
    const eventName = namespace ? `serviceaccounts:${namespace}:update` : "serviceaccounts:update";
    return EventsOn(eventName, (data: ServiceAccount[]) => setLatestServiceAccounts(data));
  }, [namespace]);
  return latestServiceAccounts;
}

import { startTransition, useEffect, useState } from "react";
import { EventsOn } from "@wailsjs/runtime/runtime";
import type { Role } from "../../api/resources";
import { mergeNamespaceScopedData } from "../../../../../shared/utils/eventMerging";

// Data-only event hook: tracks the latest pushed roles in local state.
// Called directly from role data-access hooks (useGetRoles, useGetRoleDetail, useGetRoleYAML)
// to merge event-driven data locally without cache-wide side effects.
// Pass namespaces to subscribe to the backend's namespace-scoped channels
// ("roles:{namespace}:update" for each namespace) instead of the cluster-wide "roles:update" broadcast.
export function useRolesUpdateEvents(namespaces: string[] = []): Role[] {
  const [latestRoles, setLatestRoles] = useState<Role[]>([]);
  const [prevNamespaces, setPrevNamespaces] = useState(namespaces);

  // When namespace selection changes, filter down accumulated state to only selected namespaces.
  if (JSON.stringify(prevNamespaces) !== JSON.stringify(namespaces)) {
    setPrevNamespaces(namespaces);
    if (namespaces.length > 0) {
      const namespacesSet = new Set(namespaces);
      setLatestRoles((prev) => prev.filter((role) => namespacesSet.has(role.Namespace)));
    } else {
      setLatestRoles([]);
    }
  }

  useEffect(() => {
    if (namespaces.length === 0) {
      return EventsOn("roles:update", (data: Role[]) => {
        startTransition(() => {
          setLatestRoles(data);
        });
      });
    }

    const unsubscribers: Array<() => void> = [];
    for (const ns of namespaces) {
      const eventName = `roles:${ns}:update`;
      const unsubscriber = EventsOn(eventName, (data: Role[]) => {
        startTransition(() => {
          setLatestRoles((prev) => mergeNamespaceScopedData(prev, data, ns));
        });
      });
      unsubscribers.push(unsubscriber);
    }

    return () => {
      unsubscribers.forEach((unsub) => unsub());
    };
  }, [namespaces]);
  return latestRoles;
}

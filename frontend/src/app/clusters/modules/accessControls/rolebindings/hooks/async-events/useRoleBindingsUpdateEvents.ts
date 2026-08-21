import { startTransition, useEffect, useState } from "react";
import { EventsOn } from "@wailsjs/runtime/runtime";
import type { RoleBinding } from "../../api/resources";
import { mergeNamespaceScopedData } from "../../../../../shared/utils/eventMerging";

// Data-only event hook: tracks the latest pushed rolebindings in local state.
// Called directly from rolebinding data-access hooks (useGetRoleBindings, useGetRoleBindingDetail, useGetRoleBindingYAML)
// to merge event-driven data locally without cache-wide side effects.
// Pass namespaces to subscribe to the backend's namespace-scoped channels
// ("rolebindings:{namespace}:update" for each namespace) instead of the cluster-wide "rolebindings:update" broadcast.
export function useRoleBindingsUpdateEvents(namespaces: string[] = []): RoleBinding[] {
  const [latestRoleBindings, setLatestRoleBindings] = useState<RoleBinding[]>([]);
  const [prevNamespaces, setPrevNamespaces] = useState(namespaces);

  // When namespace selection changes, filter down accumulated state to only selected namespaces.
  if (JSON.stringify(prevNamespaces) !== JSON.stringify(namespaces)) {
    setPrevNamespaces(namespaces);
    if (namespaces.length > 0) {
      const namespacesSet = new Set(namespaces);
      setLatestRoleBindings((prev) => prev.filter((rb) => namespacesSet.has(rb.Namespace)));
    } else {
      setLatestRoleBindings([]);
    }
  }

  useEffect(() => {
    if (namespaces.length === 0) {
      return EventsOn("rolebindings:update", (data: RoleBinding[]) => {
        startTransition(() => {
          setLatestRoleBindings(data);
        });
      });
    }

    const unsubscribers: Array<() => void> = [];
    for (const ns of namespaces) {
      const eventName = `rolebindings:${ns}:update`;
      const unsubscriber = EventsOn(eventName, (data: RoleBinding[]) => {
        startTransition(() => {
          setLatestRoleBindings((prev) => mergeNamespaceScopedData(prev, data, ns));
        });
      });
      unsubscribers.push(unsubscriber);
    }

    return () => {
      unsubscribers.forEach((unsub) => unsub());
    };
  }, [namespaces]);
  return latestRoleBindings;
}

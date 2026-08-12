import { startTransition, useEffect, useState } from "react";
import { EventsOn } from "@wailsjs/runtime/runtime";
import type { RoleBinding } from "../../api/resources";

// Data-only event hook: tracks the latest pushed rolebindings in local state.
// Called directly from rolebinding data-access hooks (useGetRoleBindings, useGetRoleBindingDetail, useGetRoleBindingYAML)
// to merge event-driven data locally without cache-wide side effects.
// Pass a namespace to subscribe to the backend's namespace-scoped channel
// ("rolebindings:{namespace}:update") instead of the cluster-wide "rolebindings:update" broadcast.
export function useRoleBindingsUpdateEvents(namespace = ""): RoleBinding[] {
  const [latestRoleBindings, setLatestRoleBindings] = useState<RoleBinding[]>([]);
  const [prevNamespace, setPrevNamespace] = useState(namespace);

  // Reset stale data from the previous namespace's channel before re-subscribing.
  if (namespace !== prevNamespace) {
    setPrevNamespace(namespace);
    setLatestRoleBindings([]);
  }

  useEffect(() => {
    const eventName = namespace ? `rolebindings:${namespace}:update` : "rolebindings:update";
    return EventsOn(eventName, (data: RoleBinding[]) => {
      startTransition(() => {
        setLatestRoleBindings(data);
      });
    });
  }, [namespace]);
  return latestRoleBindings;
}

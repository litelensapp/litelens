import { startTransition, useEffect, useState } from "react";
import { EventsOn } from "@wailsjs/runtime/runtime";
import type { Role } from "../../api/resources";

// Data-only event hook: tracks the latest pushed roles in local state.
// Called directly from role data-access hooks (useGetRoles, useGetRoleDetail, useGetRoleYAML)
// to merge event-driven data locally without cache-wide side effects.
// Pass a namespace to subscribe to the backend's namespace-scoped channel
// ("roles:{namespace}:update") instead of the cluster-wide "roles:update" broadcast.
export function useRolesUpdateEvents(namespace = ""): Role[] {
  const [latestRoles, setLatestRoles] = useState<Role[]>([]);
  const [prevNamespace, setPrevNamespace] = useState(namespace);

  // Reset stale data from the previous namespace's channel before re-subscribing.
  if (namespace !== prevNamespace) {
    setPrevNamespace(namespace);
    setLatestRoles([]);
  }

  useEffect(() => {
    const eventName = namespace ? `roles:${namespace}:update` : "roles:update";
    return EventsOn(eventName, (data: Role[]) => {
      startTransition(() => {
        setLatestRoles(data);
      });
    });
  }, [namespace]);
  return latestRoles;
}

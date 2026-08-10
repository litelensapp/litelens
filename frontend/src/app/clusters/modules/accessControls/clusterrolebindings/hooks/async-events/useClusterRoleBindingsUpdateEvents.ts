import { useEffect, useState } from "react";
import { EventsOn } from "@wailsjs/runtime/runtime";
import type { ClusterRoleBinding } from "../../api/resources";

export function useClusterRoleBindingsUpdateEvents(): ClusterRoleBinding[] {
  const [latestClusterRoleBindings, setLatestClusterRoleBindings] = useState<ClusterRoleBinding[]>(
    []
  );
  useEffect(() => {
    return EventsOn("clusterrolebindings:update", (data: ClusterRoleBinding[]) =>
      setLatestClusterRoleBindings(data)
    );
  }, []);
  return latestClusterRoleBindings;
}

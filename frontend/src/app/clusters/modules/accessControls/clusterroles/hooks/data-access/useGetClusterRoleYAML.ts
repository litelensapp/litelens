import { useQuery, useQueryClient } from "@tanstack/react-query";
import { useEffect, useMemo } from "react";
import { DEFAULT_QUERY_OPTIONS } from "../../../../../../shared/api/api";
import { QUERY_KEY_CLUSTER_ROLE_YAML } from "../../api/api.const";
import { GetClusterRoleYAML } from "../../api/resources";
import { useClusterRolesUpdateEvents } from "../async-events/useClusterRolesUpdateEvents";

export function useGetClusterRoleYAML(context: string, name: string, enabled = true) {
  const latestClusterRoles = useClusterRolesUpdateEvents();
  const queryClient = useQueryClient();

  const query = useQuery({
    queryKey: [QUERY_KEY_CLUSTER_ROLE_YAML, { context, name }],
    queryFn: () => GetClusterRoleYAML(name),
    ...DEFAULT_QUERY_OPTIONS,
    enabled: !!context && !!name && enabled,
  });

  const clusterRoleKeyDependency = useMemo(() => {
    const matchedClusterRole = latestClusterRoles.find((cr) => cr.Name === name);
    if (matchedClusterRole) return JSON.stringify(matchedClusterRole);
    return null;
  }, [latestClusterRoles, name]);

  useEffect(() => {
    if (clusterRoleKeyDependency) {
      queryClient.invalidateQueries({
        queryKey: [QUERY_KEY_CLUSTER_ROLE_YAML, { context, name }],
      });
    }
  }, [clusterRoleKeyDependency, context, name, queryClient]);

  return query;
}

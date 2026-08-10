import { useQuery } from "@tanstack/react-query";
import { useMemo } from "react";
import { DEFAULT_QUERY_OPTIONS } from "../../../../../../shared/api/api";
import { QUERY_KEY_CLUSTER_ROLE_DETAIL } from "../../api/api.const";
import type { ClusterRole } from "../../api/resources";
import { GetClusterRoleByName } from "../../api/resources";
import { useClusterRolesUpdateEvents } from "../async-events/useClusterRolesUpdateEvents";

export const useGetClusterRoleDetail = (context: string, name: string) => {
  const latestClusterRoles = useClusterRolesUpdateEvents();
  const query = useQuery<ClusterRole, Error>({
    queryKey: [QUERY_KEY_CLUSTER_ROLE_DETAIL, { context, name }],
    queryFn: () => GetClusterRoleByName(name),
    ...DEFAULT_QUERY_OPTIONS,
    enabled: !!context && !!name,
  });

  const mergedData = useMemo(() => {
    const matchedClusterRole = latestClusterRoles.find((cr) => cr.Name === name);
    if (matchedClusterRole) return matchedClusterRole;
    return query.data;
  }, [latestClusterRoles, query.data, name]);

  return { ...query, data: mergedData };
};

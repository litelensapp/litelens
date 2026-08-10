import { useQuery } from "@tanstack/react-query";
import { useMemo } from "react";
import { DEFAULT_QUERY_OPTIONS } from "../../../../../../shared/api/api";
import { QUERY_KEY_CLUSTER_ROLE_BINDING_DETAIL } from "../../api/api.const";
import type { ClusterRoleBinding } from "../../api/resources";
import { GetClusterRoleBindingByName } from "../../api/resources";
import { useClusterRoleBindingsUpdateEvents } from "../async-events/useClusterRoleBindingsUpdateEvents";

export const useGetClusterRoleBindingDetail = (context: string, name: string) => {
  const latestClusterRoleBindings = useClusterRoleBindingsUpdateEvents();
  const query = useQuery<ClusterRoleBinding, Error>({
    queryKey: [QUERY_KEY_CLUSTER_ROLE_BINDING_DETAIL, { context, name }],
    queryFn: () => GetClusterRoleBindingByName(name),
    ...DEFAULT_QUERY_OPTIONS,
    enabled: !!context && !!name,
  });

  const mergedData = useMemo(() => {
    const matchedClusterRoleBinding = latestClusterRoleBindings.find((crb) => crb.Name === name);
    if (matchedClusterRoleBinding) return matchedClusterRoleBinding;
    return query.data;
  }, [latestClusterRoleBindings, query.data, name]);

  return { ...query, data: mergedData };
};

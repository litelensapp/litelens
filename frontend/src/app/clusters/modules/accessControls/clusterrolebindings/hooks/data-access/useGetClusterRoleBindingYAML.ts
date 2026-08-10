import { useQuery, useQueryClient } from "@tanstack/react-query";
import { useEffect, useMemo } from "react";
import { DEFAULT_QUERY_OPTIONS } from "../../../../../../shared/api/api";
import { QUERY_KEY_CLUSTER_ROLE_BINDING_YAML } from "../../api/api.const";
import { GetClusterRoleBindingYAML } from "../../api/resources";
import { useClusterRoleBindingsUpdateEvents } from "../async-events/useClusterRoleBindingsUpdateEvents";

export function useGetClusterRoleBindingYAML(context: string, name: string, enabled = true) {
  const latestClusterRoleBindings = useClusterRoleBindingsUpdateEvents();
  const queryClient = useQueryClient();
  const query = useQuery({
    queryKey: [QUERY_KEY_CLUSTER_ROLE_BINDING_YAML, { context, name }],
    queryFn: () => GetClusterRoleBindingYAML(name),
    ...DEFAULT_QUERY_OPTIONS,
    enabled: !!context && !!name && enabled,
  });

  const matchedClusterRoleBinding = useMemo(
    () => latestClusterRoleBindings.find((crb) => crb.Name === name),
    [latestClusterRoleBindings, name]
  );
  const matchedClusterRoleBindingKey = JSON.stringify(matchedClusterRoleBinding);

  useEffect(() => {
    if (matchedClusterRoleBinding) {
      queryClient.invalidateQueries({
        queryKey: [QUERY_KEY_CLUSTER_ROLE_BINDING_YAML, { context, name }],
      });
    }
  }, [matchedClusterRoleBinding, matchedClusterRoleBindingKey, context, name, queryClient]);

  return query;
}

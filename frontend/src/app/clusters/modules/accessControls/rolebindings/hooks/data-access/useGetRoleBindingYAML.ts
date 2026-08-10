import { useQuery, useQueryClient } from "@tanstack/react-query";
import { useEffect, useMemo } from "react";
import { DEFAULT_QUERY_OPTIONS } from "../../../../../../shared/api/api";
import { QUERY_KEY_ROLE_BINDING_YAML } from "../../api/api.const";
import { GetRoleBindingYAML } from "../../api/resources";
import { useRoleBindingsUpdateEvents } from "../async-events/useRoleBindingsUpdateEvents";

export function useGetRoleBindingYAML(
  context: string,
  namespace: string,
  name: string,
  enabled = true
) {
  const queryClient = useQueryClient();
  const latestRoleBindings = useRoleBindingsUpdateEvents(namespace);

  const query = useQuery({
    queryKey: [QUERY_KEY_ROLE_BINDING_YAML, { context, namespace, name }],
    queryFn: () => GetRoleBindingYAML(namespace, name),
    ...DEFAULT_QUERY_OPTIONS,
    enabled: !!context && !!namespace && !!name && enabled,
  });

  const roleBindingKeyDependency = useMemo(() => {
    const matchedRoleBinding = latestRoleBindings.find(
      (rb) => rb.Namespace === namespace && rb.Name === name
    );
    if (matchedRoleBinding) return JSON.stringify(matchedRoleBinding);
    return null;
  }, [latestRoleBindings, namespace, name]);

  useEffect(() => {
    if (roleBindingKeyDependency)
      queryClient.invalidateQueries({
        queryKey: [QUERY_KEY_ROLE_BINDING_YAML, { context, namespace, name }],
      });
  }, [roleBindingKeyDependency, context, namespace, name, queryClient]);

  return query;
}

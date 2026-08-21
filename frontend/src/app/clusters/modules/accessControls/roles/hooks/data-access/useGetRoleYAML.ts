import { useQuery, useQueryClient } from "@tanstack/react-query";
import { useEffect, useMemo } from "react";
import { DEFAULT_QUERY_OPTIONS } from "../../../../../../shared/api/api";
import { QUERY_KEY_ROLE_YAML } from "../../api/api.const";
import { GetRoleYAML } from "../../api/resources";
import { useRolesUpdateEvents } from "../async-events/useRolesUpdateEvents";

export function useGetRoleYAML(context: string, namespace: string, name: string, enabled = true) {
  const queryClient = useQueryClient();
  const latestRoles = useRolesUpdateEvents([namespace]);

  const query = useQuery({
    queryKey: [QUERY_KEY_ROLE_YAML, { context, namespace, name }],
    queryFn: () => GetRoleYAML(namespace, name),
    ...DEFAULT_QUERY_OPTIONS,
    enabled: !!context && !!namespace && !!name && enabled,
  });

  const roleKeyDependency = useMemo(() => {
    const matchedRole = latestRoles.find((r) => r.Namespace === namespace && r.Name === name);
    if (matchedRole) return JSON.stringify(matchedRole);
    return null;
  }, [latestRoles, namespace, name]);

  useEffect(() => {
    if (roleKeyDependency)
      queryClient.invalidateQueries({
        queryKey: [QUERY_KEY_ROLE_YAML, { context, namespace, name }],
      });
  }, [roleKeyDependency, context, namespace, name, queryClient]);

  return query;
}

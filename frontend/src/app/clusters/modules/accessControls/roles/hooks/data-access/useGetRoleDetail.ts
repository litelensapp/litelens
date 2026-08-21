import { useQuery } from "@tanstack/react-query";
import { useMemo } from "react";
import { DEFAULT_QUERY_OPTIONS } from "../../../../../../shared/api/api";
import { QUERY_KEY_ROLE_DETAIL } from "../../api/api.const";
import type { Role } from "../../api/resources";
import { GetRoleByName } from "../../api/resources";
import { useRolesUpdateEvents } from "../async-events/useRolesUpdateEvents";

export const useGetRoleDetail = (context: string, namespace: string, name: string) => {
  const latestRoles = useRolesUpdateEvents([namespace]);

  const query = useQuery<Role, Error>({
    queryKey: [QUERY_KEY_ROLE_DETAIL, { context, namespace, name }],
    queryFn: () => GetRoleByName(namespace, name),
    ...DEFAULT_QUERY_OPTIONS,
    enabled: !!context && !!namespace && !!name,
  });

  const mergedData = useMemo(() => {
    const matchedRole = latestRoles.find((r) => r.Namespace === namespace && r.Name === name);
    if (matchedRole) return matchedRole;
    return query.data;
  }, [latestRoles, query.data, namespace, name]);

  return {
    ...query,
    data: mergedData,
  };
};

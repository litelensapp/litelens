import { DEFAULT_QUERY_OPTIONS } from "../../../../../../shared/api/api";
import type { UseQueryCallback } from "@litelens/design-system";
import { useMemo } from "react";
import { useQuery } from "@tanstack/react-query";
import { QUERY_KEY_ROLES } from "../../api/api.const";
import type { Role } from "../../api/resources";
import { ListRoles } from "../../api/resources";
import {
  getEffectiveNamespace,
  filterByNamespaces,
} from "../../../../../shared/utils/namespaceFiltering";
import { useRolesUpdateEvents } from "../async-events/useRolesUpdateEvents";

export const useGetRoles = (
  input: { context: string; namespaces: string[] },
  callback?: UseQueryCallback<Role[]>
) => {
  const { context, namespaces } = input;
  const effectiveNamespace = getEffectiveNamespace(namespaces);
  const latestRoles = useRolesUpdateEvents(effectiveNamespace);

  const query = useQuery<Role[], Error>({
    queryKey: [QUERY_KEY_ROLES, { context, namespaces }],
    queryFn: () => ListRoles(effectiveNamespace),
    ...DEFAULT_QUERY_OPTIONS,
    enabled: !!context,
  });

  const mergedData = useMemo(() => {
    let baseData = query.data;
    if (latestRoles.length) baseData = filterByNamespaces(latestRoles, namespaces);
    return callback?.select ? callback.select(baseData) : baseData;
  }, [latestRoles, query.data, namespaces, callback]);

  return { ...query, data: mergedData };
};

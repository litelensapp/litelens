import { DEFAULT_QUERY_OPTIONS } from "../../../../../../shared/api/api";
import type { UseQueryCallback } from "@litelens/design-system";
import { useMemo } from "react";
import { useQuery } from "@tanstack/react-query";
import { QUERY_KEY_ROLE_BINDINGS } from "../../api/api.const";
import type { RoleBinding } from "../../api/resources";
import { ListRoleBindings } from "../../api/resources";
import {
  getEffectiveNamespace,
  filterByNamespaces,
} from "../../../../../shared/utils/namespaceFiltering";
import { useRoleBindingsUpdateEvents } from "../async-events/useRoleBindingsUpdateEvents";

export const useGetRoleBindings = (
  input: { context: string; namespaces: string[] },
  callback?: UseQueryCallback<RoleBinding[]>
) => {
  const { context, namespaces } = input;
  const effectiveNamespace = getEffectiveNamespace(namespaces);
  const latestRoleBindings = useRoleBindingsUpdateEvents(effectiveNamespace);

  const query = useQuery<RoleBinding[], Error>({
    queryKey: [QUERY_KEY_ROLE_BINDINGS, { context, namespaces }],
    queryFn: () => ListRoleBindings(effectiveNamespace),
    ...DEFAULT_QUERY_OPTIONS,
    enabled: !!context,
  });

  const mergedData = useMemo(() => {
    let baseData = query.data;
    if (latestRoleBindings.length) baseData = filterByNamespaces(latestRoleBindings, namespaces);
    return callback?.select ? callback.select(baseData) : baseData;
  }, [latestRoleBindings, query.data, namespaces, callback]);

  return { ...query, data: mergedData };
};

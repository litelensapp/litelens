import { useQuery } from "@tanstack/react-query";
import { useMemo } from "react";
import { DEFAULT_QUERY_OPTIONS } from "../../../../../../shared/api/api";
import { QUERY_KEY_ROLE_BINDING_DETAIL } from "../../api/api.const";
import type { RoleBinding } from "../../api/resources";
import { GetRoleBindingByName } from "../../api/resources";
import { useRoleBindingsUpdateEvents } from "../async-events/useRoleBindingsUpdateEvents";

export const useGetRoleBindingDetail = (context: string, namespace: string, name: string) => {
  const latestRoleBindings = useRoleBindingsUpdateEvents();

  const query = useQuery<RoleBinding, Error>({
    queryKey: [QUERY_KEY_ROLE_BINDING_DETAIL, { context, namespace, name }],
    queryFn: () => GetRoleBindingByName(namespace, name),
    ...DEFAULT_QUERY_OPTIONS,
    enabled: !!context && !!namespace && !!name,
  });

  const mergedData = useMemo(() => {
    const matchedRoleBinding = latestRoleBindings.find(
      (rb) => rb.Namespace === namespace && rb.Name === name
    );
    if (matchedRoleBinding) return matchedRoleBinding;
    return query.data;
  }, [latestRoleBindings, query.data, namespace, name]);

  return {
    ...query,
    data: mergedData,
  };
};

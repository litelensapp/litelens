import type { UseQueryCallback } from "@litelens/design-system";
import { useQuery } from "@tanstack/react-query";
import { useMemo } from "react";
import { DEFAULT_QUERY_OPTIONS } from "../../../../../../shared/api/api";
import { QUERY_KEY_ROLE_BINDINGS } from "../../api/api.const";
import type { RoleBinding } from "../../api/resources";
import { ListRoleBindings } from "../../api/resources";
import { useRoleBindingsUpdateEvents } from "../async-events/useRoleBindingsUpdateEvents";

export const useGetRoleBindings = (
  input: { context: string; namespace: string },
  callback?: UseQueryCallback<RoleBinding[]>
) => {
  const { context, namespace } = input;
  const latestRoleBindings = useRoleBindingsUpdateEvents(namespace);

  const query = useQuery<RoleBinding[], Error>({
    queryKey: [QUERY_KEY_ROLE_BINDINGS, { context, namespace }],
    queryFn: () => ListRoleBindings(namespace),
    ...DEFAULT_QUERY_OPTIONS,
    enabled: !!context,
  });

  const mergedData = useMemo(() => {
    let baseData = query.data;
    if (latestRoleBindings.length)
      baseData =
        namespace === ""
          ? latestRoleBindings
          : latestRoleBindings.filter((rb) => rb.Namespace === namespace);
    return callback?.select ? callback.select(baseData) : baseData;
  }, [latestRoleBindings, query.data, namespace, callback]);

  return {
    ...query,
    data: mergedData,
  };
};

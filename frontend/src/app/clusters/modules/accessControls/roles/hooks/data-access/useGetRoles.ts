import type { UseQueryCallback } from "@litelens/design-system";
import { useQuery } from "@tanstack/react-query";
import { useMemo } from "react";
import { DEFAULT_QUERY_OPTIONS } from "../../../../../../shared/api/api";
import { QUERY_KEY_ROLES } from "../../api/api.const";
import type { Role } from "../../api/resources";
import { ListRoles } from "../../api/resources";
import { useRolesUpdateEvents } from "../async-events/useRolesUpdateEvents";

export const useGetRoles = (
  input: { context: string; namespace: string },
  callback?: UseQueryCallback<Role[]>
) => {
  const { context, namespace } = input;
  const latestRoles = useRolesUpdateEvents(namespace);

  const query = useQuery<Role[], Error>({
    queryKey: [QUERY_KEY_ROLES, { context, namespace }],
    queryFn: () => ListRoles(namespace),
    ...DEFAULT_QUERY_OPTIONS,
    enabled: !!context,
  });

  const mergedData = useMemo(() => {
    let baseData = query.data;
    if (latestRoles.length)
      baseData =
        namespace === "" ? latestRoles : latestRoles.filter((role) => role.Namespace === namespace);
    return callback?.select ? callback.select(baseData) : baseData;
  }, [latestRoles, query.data, namespace, callback]);

  return {
    ...query,
    data: mergedData,
  };
};

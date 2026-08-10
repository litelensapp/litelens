import type { UseQueryCallback } from "@litelens/design-system";
import { useQuery } from "@tanstack/react-query";
import { useMemo } from "react";
import { DEFAULT_QUERY_OPTIONS } from "../../../../../../shared/api/api";
import { QUERY_KEY_SERVICE_ACCOUNTS } from "../../api/api.const";
import type { ServiceAccount } from "../../api/resources";
import { ListServiceAccounts } from "../../api/resources";
import { useServiceAccountsUpdateEvents } from "../async-events/useServiceAccountsUpdateEvents";

export const useGetServiceAccounts = (
  input: { context: string; namespace: string },
  callback?: UseQueryCallback<ServiceAccount[]>
) => {
  const { context, namespace } = input;
  const latestServiceAccounts = useServiceAccountsUpdateEvents(namespace);

  const query = useQuery<ServiceAccount[], Error>({
    queryKey: [QUERY_KEY_SERVICE_ACCOUNTS, { context, namespace }],
    queryFn: () => ListServiceAccounts(namespace),
    ...DEFAULT_QUERY_OPTIONS,
    enabled: !!context,
  });

  const mergedData = useMemo(() => {
    let baseData = query.data;
    if (latestServiceAccounts.length)
      baseData =
        namespace === ""
          ? latestServiceAccounts
          : latestServiceAccounts.filter((sa) => sa.Namespace === namespace);
    return callback?.select ? callback.select(baseData) : baseData;
  }, [latestServiceAccounts, query.data, namespace, callback]);

  return {
    ...query,
    data: mergedData,
  };
};

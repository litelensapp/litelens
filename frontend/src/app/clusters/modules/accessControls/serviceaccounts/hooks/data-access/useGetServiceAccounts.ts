import { DEFAULT_QUERY_OPTIONS } from "../../../../../../shared/api/api";
import type { UseQueryCallback } from "@litelens/design-system";
import { useMemo } from "react";
import { useQuery } from "@tanstack/react-query";
import { QUERY_KEY_SERVICE_ACCOUNTS } from "../../api/api.const";
import type { ServiceAccount } from "../../api/resources";
import { ListServiceAccounts } from "../../api/resources";
import {
  getEffectiveNamespace,
  filterByNamespaces,
} from "../../../../../shared/utils/namespaceFiltering";
import { useServiceAccountsUpdateEvents } from "../async-events/useServiceAccountsUpdateEvents";

export const useGetServiceAccounts = (
  input: { context: string; namespaces: string[] },
  callback?: UseQueryCallback<ServiceAccount[]>
) => {
  const { context, namespaces } = input;
  const effectiveNamespace = getEffectiveNamespace(namespaces);
  const latestServiceAccounts = useServiceAccountsUpdateEvents(effectiveNamespace);

  const query = useQuery<ServiceAccount[], Error>({
    queryKey: [QUERY_KEY_SERVICE_ACCOUNTS, { context, namespaces }],
    queryFn: () => ListServiceAccounts(effectiveNamespace),
    ...DEFAULT_QUERY_OPTIONS,
    enabled: !!context,
  });

  const mergedData = useMemo(() => {
    let baseData = query.data;
    if (latestServiceAccounts.length)
      baseData = filterByNamespaces(latestServiceAccounts, namespaces);
    return callback?.select ? callback.select(baseData) : baseData;
  }, [latestServiceAccounts, query.data, namespaces, callback]);

  return { ...query, data: mergedData };
};

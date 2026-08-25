import { useQuery } from "@tanstack/react-query";
import { useMemo } from "react";
import { DEFAULT_QUERY_OPTIONS } from "../../../../../../shared/api/api";
import { QUERY_KEY_SERVICE_ACCOUNT_DETAIL } from "../../api/api.const";
import type { ServiceAccount } from "../../api/resources";
import { GetServiceAccountByName } from "../../api/resources";
import { useServiceAccountsUpdateEvents } from "../async-events/useServiceAccountsUpdateEvents";

export const useGetServiceAccountDetail = (context: string, namespace: string, name: string) => {
  const latestServiceAccounts = useServiceAccountsUpdateEvents();

  const query = useQuery<ServiceAccount, Error>({
    queryKey: [QUERY_KEY_SERVICE_ACCOUNT_DETAIL, { context, namespace, name }],
    queryFn: () => GetServiceAccountByName(namespace, name),
    ...DEFAULT_QUERY_OPTIONS,
    enabled: !!context && !!namespace && !!name,
  });

  const mergedData = useMemo(() => {
    const matchedServiceAccount = latestServiceAccounts.find(
      (sa) => sa.Namespace === namespace && sa.Name === name
    );
    if (matchedServiceAccount) return matchedServiceAccount;
    return query.data;
  }, [latestServiceAccounts, query.data, namespace, name]);

  return {
    ...query,
    data: mergedData,
  };
};

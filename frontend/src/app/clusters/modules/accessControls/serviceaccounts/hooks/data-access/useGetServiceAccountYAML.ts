import { useQuery, useQueryClient } from "@tanstack/react-query";
import { useEffect, useMemo } from "react";
import { DEFAULT_QUERY_OPTIONS } from "../../../../../../shared/api/api";
import { QUERY_KEY_SERVICE_ACCOUNT_YAML } from "../../api/api.const";
import { GetServiceAccountYAML } from "../../api/resources";
import { useServiceAccountsUpdateEvents } from "../async-events/useServiceAccountsUpdateEvents";

export function useGetServiceAccountYAML(
  context: string,
  namespace: string,
  name: string,
  enabled = true
) {
  const queryClient = useQueryClient();
  const latestServiceAccounts = useServiceAccountsUpdateEvents(namespace);

  const query = useQuery({
    queryKey: [QUERY_KEY_SERVICE_ACCOUNT_YAML, { context, namespace, name }],
    queryFn: () => GetServiceAccountYAML(namespace, name),
    ...DEFAULT_QUERY_OPTIONS,
    enabled: !!context && !!namespace && !!name && enabled,
  });

  const serviceAccountKeyDependency = useMemo(() => {
    const matchedServiceAccount = latestServiceAccounts.find(
      (sa) => sa.Namespace === namespace && sa.Name === name
    );
    if (matchedServiceAccount) return JSON.stringify(matchedServiceAccount);
    return null;
  }, [latestServiceAccounts, namespace, name]);

  useEffect(() => {
    if (serviceAccountKeyDependency)
      queryClient.invalidateQueries({
        queryKey: [QUERY_KEY_SERVICE_ACCOUNT_YAML, { context, namespace, name }],
      });
  }, [serviceAccountKeyDependency, context, namespace, name, queryClient]);

  return query;
}

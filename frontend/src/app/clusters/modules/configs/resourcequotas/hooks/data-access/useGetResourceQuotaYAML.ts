import { DEFAULT_QUERY_OPTIONS } from "../../../../../../shared/api/api";
import { QUERY_KEY_RESOURCE_QUOTA_YAML } from "../../api/api.const";
import { GetResourceQuotaYAML } from "../../api/resources";
import { useQuery, useQueryClient } from "@tanstack/react-query";
import { useEffect, useMemo } from "react";
import { useResourceQuotasUpdateEvents } from "../async-events/useResourceQuotasUpdateEvents";

export function useGetResourceQuotaYAML(
  context: string,
  namespace: string,
  name: string,
  enabled = true
) {
  const queryClient = useQueryClient();
  const latestResourceQuotas = useResourceQuotasUpdateEvents(namespace);
  const query = useQuery({
    queryKey: [QUERY_KEY_RESOURCE_QUOTA_YAML, { context, namespace, name }],
    queryFn: () => GetResourceQuotaYAML(namespace, name),
    ...DEFAULT_QUERY_OPTIONS,
    enabled: !!context && !!namespace && !!name && enabled,
  });

  const matchedResourceQuota = useMemo(
    () => latestResourceQuotas.find((rq) => rq.Namespace === namespace && rq.Name === name),
    [latestResourceQuotas, namespace, name]
  );

  useEffect(() => {
    if (matchedResourceQuota) {
      queryClient.invalidateQueries({
        queryKey: [QUERY_KEY_RESOURCE_QUOTA_YAML, { context, namespace, name }],
      });
    }
  }, [matchedResourceQuota, context, namespace, name, queryClient]);

  return query;
}

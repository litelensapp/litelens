import { useEffect, useMemo } from "react";
import { useQuery, useQueryClient } from "@tanstack/react-query";
import { QUERY_KEY_RESOURCE_QUOTA_DETAIL } from "../../api/api.const";
import { DEFAULT_QUERY_OPTIONS } from "../../../../../../shared/api/api";
import type { ResourceQuotaDetail } from "../../api/resources";
import { GetResourceQuotaByName } from "../../api/resources";
import { useResourceQuotasUpdateEvents } from "../async-events/useResourceQuotasUpdateEvents";

export const useGetResourceQuotaDetail = (context: string, namespace: string, name: string) => {
  const queryClient = useQueryClient();
  const latestResourceQuotas = useResourceQuotasUpdateEvents([namespace]);
  const query = useQuery<ResourceQuotaDetail, Error>({
    queryKey: [QUERY_KEY_RESOURCE_QUOTA_DETAIL, { context, namespace, name }],
    queryFn: () => GetResourceQuotaByName(namespace, name),
    ...DEFAULT_QUERY_OPTIONS,
    enabled: !!context && !!namespace && !!name,
  });

  const matchedResourceQuota = useMemo(
    () => latestResourceQuotas.find((rq) => rq.Namespace === namespace && rq.Name === name),
    [latestResourceQuotas, namespace, name]
  );

  useEffect(() => {
    if (matchedResourceQuota) {
      queryClient.invalidateQueries({
        queryKey: [QUERY_KEY_RESOURCE_QUOTA_DETAIL, { context, namespace, name }],
      });
    }
  }, [matchedResourceQuota, context, namespace, name, queryClient]);

  return query;
};

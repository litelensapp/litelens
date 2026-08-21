import { DEFAULT_QUERY_OPTIONS } from "../../../../../../shared/api/api";
import type { UseQueryCallback } from "@litelens/core";
import { useMemo } from "react";
import { useQuery } from "@tanstack/react-query";
import { QUERY_KEY_RESOURCE_QUOTAS } from "../../api/api.const";
import type { ResourceQuota } from "../../api/resources";
import { ListResourceQuotas } from "../../api/resources";
import { filterByNamespaces } from "../../../../../shared/utils/namespaceFiltering";
import { useResourceQuotasUpdateEvents } from "../async-events/useResourceQuotasUpdateEvents";

export const useGetResourceQuotas = (
  input: { context: string; namespaces: string[] },
  callback?: UseQueryCallback<ResourceQuota[]>
) => {
  const { context, namespaces } = input;
  const latestResourceQuotas = useResourceQuotasUpdateEvents(namespaces);

  const query = useQuery<ResourceQuota[], Error>({
    queryKey: [QUERY_KEY_RESOURCE_QUOTAS, { context, namespaces }],
    queryFn: () => ListResourceQuotas(namespaces),
    ...DEFAULT_QUERY_OPTIONS,
    enabled: !!context,
  });

  const mergedData = useMemo(() => {
    let baseData = query.data;
    if (latestResourceQuotas.length)
      baseData = filterByNamespaces(latestResourceQuotas, namespaces);
    return callback?.select ? callback.select(baseData) : baseData;
  }, [latestResourceQuotas, query.data, namespaces, callback]);

  return { ...query, data: mergedData };
};

import { DEFAULT_QUERY_OPTIONS } from "../../../../../../shared/api/api";
import type { UseQueryCallback } from "@litelens/design-system";
import { useMemo } from "react";
import { useQuery } from "@tanstack/react-query";
import { QUERY_KEY_RESOURCE_QUOTAS } from "../../api/api.const";
import type { ResourceQuota } from "../../api/resources";
import { ListResourceQuotas } from "../../api/resources";
import { useResourceQuotasUpdateEvents } from "../async-events/useResourceQuotasUpdateEvents";

export const useGetResourceQuotas = (
  input: { context: string; namespace: string },
  callback?: UseQueryCallback<ResourceQuota[]>
) => {
  const { context, namespace } = input;
  const latestResourceQuotas = useResourceQuotasUpdateEvents(namespace);
  const query = useQuery<ResourceQuota[], Error>({
    queryKey: [QUERY_KEY_RESOURCE_QUOTAS, { context, namespace }],
    queryFn: () => ListResourceQuotas(namespace),
    ...DEFAULT_QUERY_OPTIONS,
    enabled: !!context,
  });

  const mergedData = useMemo(() => {
    let baseData = query.data;
    if (latestResourceQuotas.length)
      baseData =
        namespace === ""
          ? latestResourceQuotas
          : latestResourceQuotas.filter((rq) => rq.Namespace === namespace);
    return callback?.select ? callback.select(baseData) : baseData;
  }, [latestResourceQuotas, query.data, namespace, callback]);

  return { ...query, data: mergedData };
};

import { DEFAULT_QUERY_OPTIONS } from "../../../../../../shared/api/api";
import type { UseQueryCallback } from "@litelens/core";
import { useMemo } from "react";
import { useQuery } from "@tanstack/react-query";
import { QUERY_KEY_SERVICES } from "../../api/api.const";
import type { Service } from "../../api/resources";
import { ListServices } from "../../api/resources";
import { filterByNamespaces } from "../../../../../shared/utils/namespaceFiltering";
import { useServicesUpdateEvents } from "../async-events/useServicesUpdateEvents";

export const useGetServices = (
  input: { context: string; namespaces: string[] },
  callback?: UseQueryCallback<Service[]>
) => {
  const { context, namespaces } = input;
  const latestServices = useServicesUpdateEvents(namespaces);

  const query = useQuery<Service[], Error>({
    queryKey: [QUERY_KEY_SERVICES, { context, namespaces }],
    queryFn: () => ListServices(namespaces),
    ...DEFAULT_QUERY_OPTIONS,
    enabled: !!context,
  });

  const mergedData = useMemo(() => {
    let baseData = query.data;
    if (latestServices.length) baseData = filterByNamespaces(latestServices, namespaces);
    return callback?.select ? callback.select(baseData) : baseData;
  }, [latestServices, query.data, namespaces, callback]);

  return { ...query, data: mergedData };
};

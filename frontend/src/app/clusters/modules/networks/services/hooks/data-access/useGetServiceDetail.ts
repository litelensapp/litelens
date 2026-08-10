import { DEFAULT_QUERY_OPTIONS } from "../../../../../../shared/api/api";
import { useQuery } from "@tanstack/react-query";
import { useMemo } from "react";
import { QUERY_KEY_SERVICE_DETAIL } from "../../api/api.const";
import type { Service } from "../../api/resources";
import { GetServiceByName } from "../../api/resources";
import { useServicesUpdateEvents } from "../async-events/useServicesUpdateEvents";

export const useGetServiceDetail = (context: string, namespace: string, name: string) => {
  const latestServices = useServicesUpdateEvents();

  const query = useQuery<Service, Error>({
    queryKey: [QUERY_KEY_SERVICE_DETAIL, { context, namespace, name }],
    queryFn: () => GetServiceByName(namespace, name),
    ...DEFAULT_QUERY_OPTIONS,
    enabled: !!context && !!namespace && !!name,
  });

  // Merge event-driven data: prefer matched service from latest event if available.
  const mergedData = useMemo(() => {
    const matchedService = latestServices.find((s) => s.Namespace === namespace && s.Name === name);
    if (matchedService) return matchedService;
    return query.data;
  }, [latestServices, query.data, namespace, name]);

  return {
    ...query,
    data: mergedData,
  };
};

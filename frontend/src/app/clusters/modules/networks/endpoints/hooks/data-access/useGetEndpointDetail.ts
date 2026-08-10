import { DEFAULT_QUERY_OPTIONS } from "../../../../../../shared/api/api";
import { useQuery } from "@tanstack/react-query";
import { useMemo } from "react";
import { QUERY_KEY_ENDPOINT_DETAIL } from "../../api/api.const";
import type { Endpoint } from "../../api/resources";
import { GetEndpointByName } from "../../api/resources";
import { useEndpointsUpdateEvents } from "../async-events/useEndpointsUpdateEvents";

export const useGetEndpointDetail = (context: string, namespace: string, name: string) => {
  const latestEndpoints = useEndpointsUpdateEvents();

  const query = useQuery<Endpoint, Error>({
    queryKey: [QUERY_KEY_ENDPOINT_DETAIL, { context, namespace, name }],
    queryFn: () => GetEndpointByName(namespace, name),
    ...DEFAULT_QUERY_OPTIONS,
    enabled: !!context && !!namespace && !!name,
  });

  // Merge event-driven data: prefer matched endpoint from latest event if available.
  const mergedData = useMemo(() => {
    const matchedEndpoint = latestEndpoints.find(
      (e) => e.Namespace === namespace && e.Name === name
    );
    if (matchedEndpoint) return matchedEndpoint;
    return query.data;
  }, [latestEndpoints, query.data, namespace, name]);

  return {
    ...query,
    data: mergedData,
  };
};

import { DEFAULT_QUERY_OPTIONS } from "../../../../../../shared/api/api";
import type { UseQueryCallback } from "@litelens/design-system";
import { useQuery } from "@tanstack/react-query";
import { useMemo } from "react";
import { QUERY_KEY_ENDPOINTS } from "../../api/api.const";
import type { Endpoint } from "../../api/resources";
import { ListEndpoints } from "../../api/resources";
import { useEndpointsUpdateEvents } from "../async-events/useEndpointsUpdateEvents";

export const useGetEndpoints = (
  input: { context: string; namespace: string },
  callback?: UseQueryCallback<Endpoint[]>
) => {
  const { context, namespace } = input;
  const latestEndpoints = useEndpointsUpdateEvents();

  const query = useQuery<Endpoint[], Error>({
    queryKey: [QUERY_KEY_ENDPOINTS, { context, namespace }],
    queryFn: () => ListEndpoints(namespace),
    ...DEFAULT_QUERY_OPTIONS,
    enabled: !!context,
  });

  // Merge event-driven data locally: prefer event-filtered endpoints over fetched data if available.
  // Filter cluster-wide event list to this hook's namespace (or include all if namespace === "").
  const mergedData = useMemo(() => {
    let baseData = query.data;
    if (latestEndpoints.length)
      baseData =
        namespace === ""
          ? latestEndpoints
          : latestEndpoints.filter((ep) => ep.Namespace === namespace);
    return callback?.select ? callback.select(baseData) : baseData;
  }, [latestEndpoints, query.data, namespace, callback]);

  return {
    ...query,
    data: mergedData,
  };
};

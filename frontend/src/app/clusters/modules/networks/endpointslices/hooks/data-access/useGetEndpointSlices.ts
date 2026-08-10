import { DEFAULT_QUERY_OPTIONS } from "../../../../../../shared/api/api";
import type { UseQueryCallback } from "@litelens/design-system";
import { useQuery } from "@tanstack/react-query";
import { useMemo } from "react";
import { QUERY_KEY_ENDPOINT_SLICES } from "../../api/api.const";
import type { EndpointSlice } from "../../api/resources";
import { ListEndpointSlices } from "../../api/resources";
import { useEndpointSlicesUpdateEvents } from "../async-events/useEndpointSlicesUpdateEvents";

export const useGetEndpointSlices = (
  input: { context: string; namespace: string },
  callback?: UseQueryCallback<EndpointSlice[]>
) => {
  const { context, namespace } = input;
  const latestEndpointSlices = useEndpointSlicesUpdateEvents();

  const query = useQuery<EndpointSlice[], Error>({
    queryKey: [QUERY_KEY_ENDPOINT_SLICES, { context, namespace }],
    queryFn: () => ListEndpointSlices(namespace),
    ...DEFAULT_QUERY_OPTIONS,
    enabled: !!context,
  });

  // Merge event-driven data locally: prefer event-filtered endpoint slices over fetched data if available.
  // Filter cluster-wide event list to this hook's namespace (or include all if namespace === "").
  const mergedData = useMemo(() => {
    let baseData = query.data;
    if (latestEndpointSlices.length)
      baseData =
        namespace === ""
          ? latestEndpointSlices
          : latestEndpointSlices.filter((eps) => eps.Namespace === namespace);
    return callback?.select ? callback.select(baseData) : baseData;
  }, [latestEndpointSlices, query.data, namespace, callback]);

  return {
    ...query,
    data: mergedData,
  };
};

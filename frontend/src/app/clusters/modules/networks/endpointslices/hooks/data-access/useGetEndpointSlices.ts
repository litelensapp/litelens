import { DEFAULT_QUERY_OPTIONS } from "../../../../../../shared/api/api";
import type { UseQueryCallback } from "@litelens/core";
import { useMemo } from "react";
import { useQuery } from "@tanstack/react-query";
import { QUERY_KEY_ENDPOINT_SLICES } from "../../api/api.const";
import type { EndpointSlice } from "../../api/resources";
import { ListEndpointSlices } from "../../api/resources";
import { filterByNamespaces } from "../../../../../shared/utils/namespaceFiltering";
import { useEndpointSlicesUpdateEvents } from "../async-events/useEndpointSlicesUpdateEvents";

export const useGetEndpointSlices = (
  input: { context: string; namespaces: string[] },
  callback?: UseQueryCallback<EndpointSlice[]>
) => {
  const { context, namespaces } = input;
  const latestEndpointSlices = useEndpointSlicesUpdateEvents(namespaces);

  const query = useQuery<EndpointSlice[], Error>({
    queryKey: [QUERY_KEY_ENDPOINT_SLICES, { context, namespaces }],
    queryFn: () => ListEndpointSlices(namespaces),
    ...DEFAULT_QUERY_OPTIONS,
    enabled: !!context,
  });

  const mergedData = useMemo(() => {
    let baseData = query.data;
    if (latestEndpointSlices.length)
      baseData = filterByNamespaces(latestEndpointSlices, namespaces);
    return callback?.select ? callback.select(baseData) : baseData;
  }, [latestEndpointSlices, query.data, namespaces, callback]);

  return { ...query, data: mergedData };
};

import { DEFAULT_QUERY_OPTIONS } from "../../../../../../shared/api/api";
import type { UseQueryCallback } from "@litelens/core";
import { useMemo } from "react";
import { useQuery } from "@tanstack/react-query";
import { QUERY_KEY_ENDPOINTS } from "../../api/api.const";
import type { Endpoint } from "../../api/resources";
import { ListEndpoints } from "../../api/resources";
import {
  getEffectiveNamespace,
  filterByNamespaces,
} from "../../../../../shared/utils/namespaceFiltering";
import { useEndpointsUpdateEvents } from "../async-events/useEndpointsUpdateEvents";

export const useGetEndpoints = (
  input: { context: string; namespaces: string[] },
  callback?: UseQueryCallback<Endpoint[]>
) => {
  const { context, namespaces } = input;
  const effectiveNamespace = getEffectiveNamespace(namespaces);
  const latestEndpoints = useEndpointsUpdateEvents();

  const query = useQuery<Endpoint[], Error>({
    queryKey: [QUERY_KEY_ENDPOINTS, { context, namespaces }],
    queryFn: () => ListEndpoints(effectiveNamespace),
    ...DEFAULT_QUERY_OPTIONS,
    enabled: !!context,
  });

  const mergedData = useMemo(() => {
    let baseData = query.data;
    if (latestEndpoints.length) baseData = filterByNamespaces(latestEndpoints, namespaces);
    return callback?.select ? callback.select(baseData) : baseData;
  }, [latestEndpoints, query.data, namespaces, callback]);

  return { ...query, data: mergedData };
};

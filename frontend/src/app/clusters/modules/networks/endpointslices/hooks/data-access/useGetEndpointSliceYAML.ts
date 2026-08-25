import { DEFAULT_QUERY_OPTIONS } from "../../../../../../shared/api/api";
import { QUERY_KEY_ENDPOINT_SLICE_YAML } from "../../api/api.const";
import { GetEndpointSliceYAML } from "../../api/resources";
import { useQuery, useQueryClient } from "@tanstack/react-query";
import { useEffect, useMemo } from "react";
import { useEndpointSlicesUpdateEvents } from "../async-events/useEndpointSlicesUpdateEvents";

export function useGetEndpointSliceYAML(
  context: string,
  namespace: string,
  name: string,
  enabled = true
) {
  const queryClient = useQueryClient();
  const latestEndpointSlices = useEndpointSlicesUpdateEvents();

  const query = useQuery({
    queryKey: [QUERY_KEY_ENDPOINT_SLICE_YAML, { context, namespace, name }],
    queryFn: () => GetEndpointSliceYAML(namespace, name),
    ...DEFAULT_QUERY_OPTIONS,
    enabled: !!context && !!namespace && !!name && enabled,
  });

  // Invalidate YAML cache for this endpoint slice when a matching endpoint slice update is received.
  // Use a stable derived value (serialized endpoint slice key) as dependency to avoid invalidating
  // on every unrelated event churn.
  const endpointSliceKeyDependency = useMemo(() => {
    const matchedEndpointSlice = latestEndpointSlices.find(
      (eps) => eps.Namespace === namespace && eps.Name === name
    );
    // Serialize the endpoint slice to a stable string: changes only when the endpoint slice's content meaningfully changes.
    if (matchedEndpointSlice) return JSON.stringify(matchedEndpointSlice);
    return null;
  }, [latestEndpointSlices, namespace, name]);

  useEffect(() => {
    if (endpointSliceKeyDependency)
      queryClient.invalidateQueries({
        queryKey: [QUERY_KEY_ENDPOINT_SLICE_YAML, { context, namespace, name }],
      });
  }, [endpointSliceKeyDependency, context, namespace, name, queryClient]);

  return query;
}

import { DEFAULT_QUERY_OPTIONS } from "../../../../../../shared/api/api";
import { QUERY_KEY_ENDPOINT_YAML } from "../../api/api.const";
import { GetEndpointYAML } from "../../api/resources";
import { useQuery, useQueryClient } from "@tanstack/react-query";
import { useEffect, useMemo } from "react";
import { useEndpointsUpdateEvents } from "../async-events/useEndpointsUpdateEvents";

export function useGetEndpointYAML(
  context: string,
  namespace: string,
  name: string,
  enabled = true
) {
  const queryClient = useQueryClient();
  const latestEndpoints = useEndpointsUpdateEvents();

  const query = useQuery({
    queryKey: [QUERY_KEY_ENDPOINT_YAML, { context, namespace, name }],
    queryFn: () => GetEndpointYAML(namespace, name),
    ...DEFAULT_QUERY_OPTIONS,
    enabled: !!context && !!namespace && !!name && enabled,
  });

  // Invalidate YAML cache for this endpoint when a matching endpoint update is received.
  // Use a stable derived value (serialized endpoint key) as dependency to avoid invalidating
  // on every unrelated event churn.
  const endpointKeyDependency = useMemo(() => {
    const matchedEndpoint = latestEndpoints.find(
      (e) => e.Namespace === namespace && e.Name === name
    );
    // Serialize the endpoint to a stable string: changes only when the endpoint's content meaningfully changes.
    if (matchedEndpoint) return JSON.stringify(matchedEndpoint);
    return null;
  }, [latestEndpoints, namespace, name]);

  useEffect(() => {
    if (endpointKeyDependency)
      queryClient.invalidateQueries({
        queryKey: [QUERY_KEY_ENDPOINT_YAML, { context, namespace, name }],
      });
  }, [endpointKeyDependency, context, namespace, name, queryClient]);

  return query;
}

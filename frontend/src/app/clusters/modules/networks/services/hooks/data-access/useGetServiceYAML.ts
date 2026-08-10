import { DEFAULT_QUERY_OPTIONS } from "../../../../../../shared/api/api";
import { QUERY_KEY_SERVICE_YAML } from "../../api/api.const";
import { GetServiceYAML } from "../../api/resources";
import { useQuery, useQueryClient } from "@tanstack/react-query";
import { useEffect, useMemo } from "react";
import { useServicesUpdateEvents } from "../async-events/useServicesUpdateEvents";

export function useGetServiceYAML(
  context: string,
  namespace: string,
  name: string,
  enabled = true
) {
  const queryClient = useQueryClient();
  const latestServices = useServicesUpdateEvents();

  const query = useQuery({
    queryKey: [QUERY_KEY_SERVICE_YAML, { context, namespace, name }],
    queryFn: () => GetServiceYAML(namespace, name),
    ...DEFAULT_QUERY_OPTIONS,
    enabled: !!context && !!namespace && !!name && enabled,
  });

  // Invalidate YAML cache for this service when a matching service update is received.
  // Use a stable derived value (serialized service key) as dependency to avoid invalidating
  // on every unrelated event churn.
  const serviceKeyDependency = useMemo(() => {
    const matchedService = latestServices.find((s) => s.Namespace === namespace && s.Name === name);
    // Serialize the service to a stable string: changes only when the service's content meaningfully changes.
    if (matchedService) return JSON.stringify(matchedService);
    return null;
  }, [latestServices, namespace, name]);

  useEffect(() => {
    if (serviceKeyDependency)
      queryClient.invalidateQueries({
        queryKey: [QUERY_KEY_SERVICE_YAML, { context, namespace, name }],
      });
  }, [serviceKeyDependency, context, namespace, name, queryClient]);

  return query;
}

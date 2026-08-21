import { DEFAULT_QUERY_OPTIONS } from "../../../../../../shared/api/api";
import { QUERY_KEY_INGRESS_YAML } from "../../api/api.const";
import { GetIngressYAML } from "../../api/resources";
import { useQuery, useQueryClient } from "@tanstack/react-query";
import { useEffect, useMemo } from "react";
import { useIngressesUpdateEvents } from "../async-events/useIngressesUpdateEvents";

export function useGetIngressYAML(
  context: string,
  namespace: string,
  name: string,
  enabled = true
) {
  const queryClient = useQueryClient();
  const latestIngresses = useIngressesUpdateEvents([namespace]);

  const query = useQuery({
    queryKey: [QUERY_KEY_INGRESS_YAML, { context, namespace, name }],
    queryFn: () => GetIngressYAML(namespace, name),
    ...DEFAULT_QUERY_OPTIONS,
    enabled: !!context && !!namespace && !!name && enabled,
  });

  // Invalidate YAML cache for this ingress when a matching ingress update is received.
  // Use a stable derived value (serialized ingress key) as dependency to avoid invalidating
  // on every unrelated event churn.
  const ingressKeyDependency = useMemo(() => {
    const matchedIngress = latestIngresses.find(
      (ing) => ing.Namespace === namespace && ing.Name === name
    );
    // Serialize the ingress to a stable string: changes only when the ingress's content meaningfully changes.
    if (matchedIngress) return JSON.stringify(matchedIngress);
    return null;
  }, [latestIngresses, namespace, name]);

  useEffect(() => {
    if (ingressKeyDependency)
      queryClient.invalidateQueries({
        queryKey: [QUERY_KEY_INGRESS_YAML, { context, namespace, name }],
      });
  }, [ingressKeyDependency, context, namespace, name, queryClient]);

  return query;
}

import { DEFAULT_QUERY_OPTIONS } from "../../../../../../shared/api/api";
import { QUERY_KEY_NETWORK_POLICY_YAML } from "../../api/api.const";
import { GetNetworkPolicyYAML } from "../../api/resources";
import { useQuery, useQueryClient } from "@tanstack/react-query";
import { useEffect, useMemo } from "react";
import { useNetworkPoliciesUpdateEvents } from "../async-events/useNetworkPoliciesUpdateEvents";

export function useGetNetworkPolicyYAML(
  context: string,
  namespace: string,
  name: string,
  enabled = true
) {
  const queryClient = useQueryClient();
  const latestNetworkPolicies = useNetworkPoliciesUpdateEvents();

  const query = useQuery({
    queryKey: [QUERY_KEY_NETWORK_POLICY_YAML, { context, namespace, name }],
    queryFn: () => GetNetworkPolicyYAML(namespace, name),
    ...DEFAULT_QUERY_OPTIONS,
    enabled: !!context && !!namespace && !!name && enabled,
  });

  const npKeyDependency = useMemo(() => {
    const matchedNetworkPolicy = latestNetworkPolicies.find(
      (np) => np.Namespace === namespace && np.Name === name
    );
    if (matchedNetworkPolicy) return JSON.stringify(matchedNetworkPolicy);
    return null;
  }, [latestNetworkPolicies, namespace, name]);

  useEffect(() => {
    if (npKeyDependency)
      queryClient.invalidateQueries({
        queryKey: [QUERY_KEY_NETWORK_POLICY_YAML, { context, namespace, name }],
      });
  }, [npKeyDependency, context, namespace, name, queryClient]);

  return query;
}

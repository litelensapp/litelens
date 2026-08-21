import { DEFAULT_QUERY_OPTIONS } from "../../../../../../shared/api/api";
import { useQuery, useQueryClient } from "@tanstack/react-query";
import { useEffect, useMemo } from "react";
import { QUERY_KEY_NETWORK_POLICY_DETAIL } from "../../api/api.const";
import type { NetworkPolicyDetail } from "../../api/resources";
import { GetNetworkPolicyByName } from "../../api/resources";
import { useNetworkPoliciesUpdateEvents } from "../async-events/useNetworkPoliciesUpdateEvents";

export const useGetNetworkPolicyDetail = (context: string, namespace: string, name: string) => {
  const queryClient = useQueryClient();
  const latestNetworkPolicies = useNetworkPoliciesUpdateEvents([namespace]);

  const query = useQuery<NetworkPolicyDetail, Error>({
    queryKey: [QUERY_KEY_NETWORK_POLICY_DETAIL, { context, namespace, name }],
    queryFn: () => GetNetworkPolicyByName(namespace, name),
    ...DEFAULT_QUERY_OPTIONS,
    enabled: !!context && !!namespace && !!name,
  });

  // The pushed event carries the lighter list DTO, not the full detail shape — invalidate
  // to refetch the detail rather than overwriting the cache with a mismatched shape.
  const networkPolicyKeyDependency = useMemo(() => {
    const matchedNetworkPolicy = latestNetworkPolicies.find(
      (np) => np.Namespace === namespace && np.Name === name
    );
    return matchedNetworkPolicy ? JSON.stringify(matchedNetworkPolicy) : null;
  }, [latestNetworkPolicies, namespace, name]);

  useEffect(() => {
    if (networkPolicyKeyDependency)
      queryClient.invalidateQueries({
        queryKey: [QUERY_KEY_NETWORK_POLICY_DETAIL, { context, namespace, name }],
      });
  }, [networkPolicyKeyDependency, context, namespace, name, queryClient]);

  return query;
};

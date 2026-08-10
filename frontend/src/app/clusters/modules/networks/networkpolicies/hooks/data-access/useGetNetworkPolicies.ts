import { DEFAULT_QUERY_OPTIONS } from "../../../../../../shared/api/api";
import type { UseQueryCallback } from "@litelens/design-system";
import { useQuery } from "@tanstack/react-query";
import { useMemo } from "react";
import { QUERY_KEY_NETWORK_POLICIES } from "../../api/api.const";
import type { NetworkPolicy } from "../../api/resources";
import { ListNetworkPolicies } from "../../api/resources";
import { useNetworkPoliciesUpdateEvents } from "../async-events/useNetworkPoliciesUpdateEvents";

export const useGetNetworkPolicies = (
  input: { context: string; namespace: string },
  callback?: UseQueryCallback<NetworkPolicy[]>
) => {
  const { context, namespace } = input;
  const latestNetworkPolicies = useNetworkPoliciesUpdateEvents();

  const query = useQuery<NetworkPolicy[], Error>({
    queryKey: [QUERY_KEY_NETWORK_POLICIES, { context, namespace }],
    queryFn: () => ListNetworkPolicies(namespace),
    ...DEFAULT_QUERY_OPTIONS,
    enabled: !!context,
  });

  const mergedData = useMemo(() => {
    let baseData = query.data;
    if (latestNetworkPolicies.length)
      baseData =
        namespace === ""
          ? latestNetworkPolicies
          : latestNetworkPolicies.filter((np) => np.Namespace === namespace);
    return callback?.select ? callback.select(baseData) : baseData;
  }, [latestNetworkPolicies, query.data, namespace, callback]);

  return { ...query, data: mergedData };
};

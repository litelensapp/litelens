import { DEFAULT_QUERY_OPTIONS } from "../../../../../../shared/api/api";
import type { UseQueryCallback } from "@litelens/core";
import { useMemo } from "react";
import { useQuery } from "@tanstack/react-query";
import { QUERY_KEY_NETWORK_POLICIES } from "../../api/api.const";
import type { NetworkPolicy } from "../../api/resources";
import { ListNetworkPolicies } from "../../api/resources";
import {
  getEffectiveNamespace,
  filterByNamespaces,
} from "../../../../../shared/utils/namespaceFiltering";
import { useNetworkPoliciesUpdateEvents } from "../async-events/useNetworkPoliciesUpdateEvents";

export const useGetNetworkPolicies = (
  input: { context: string; namespaces: string[] },
  callback?: UseQueryCallback<NetworkPolicy[]>
) => {
  const { context, namespaces } = input;
  const effectiveNamespace = getEffectiveNamespace(namespaces);
  const latestNetworkPolicies = useNetworkPoliciesUpdateEvents();

  const query = useQuery<NetworkPolicy[], Error>({
    queryKey: [QUERY_KEY_NETWORK_POLICIES, { context, namespaces }],
    queryFn: () => ListNetworkPolicies(effectiveNamespace),
    ...DEFAULT_QUERY_OPTIONS,
    enabled: !!context,
  });

  const mergedData = useMemo(() => {
    let baseData = query.data;
    if (latestNetworkPolicies.length)
      baseData = filterByNamespaces(latestNetworkPolicies, namespaces);
    return callback?.select ? callback.select(baseData) : baseData;
  }, [latestNetworkPolicies, query.data, namespaces, callback]);

  return { ...query, data: mergedData };
};

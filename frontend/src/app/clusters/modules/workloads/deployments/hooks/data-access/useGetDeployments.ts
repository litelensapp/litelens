import { DEFAULT_QUERY_OPTIONS } from "../../../../../../shared/api/api";
import type { UseQueryCallback } from "@litelens/core";
import { useQuery } from "@tanstack/react-query";
import { useMemo } from "react";
import { QUERY_KEY_DEPLOYMENTS } from "../../api/api.const";
import type { Deployment } from "../../api/resources";
import { ListDeployments } from "../../api/resources";
import { useDeploymentsUpdateEvents } from "../async-events/useDeploymentsUpdateEvents";
import {
  getEffectiveNamespace,
  filterByNamespaces,
} from "../../../../../shared/utils/namespaceFiltering";

export const useGetDeployments = (
  input: { context: string; namespaces: string[] },
  callback?: UseQueryCallback<Deployment[]>
) => {
  const { context, namespaces } = input;
  const effectiveNamespace = getEffectiveNamespace(namespaces);
  const latestDeployments = useDeploymentsUpdateEvents();

  const query = useQuery<Deployment[], Error>({
    queryKey: [QUERY_KEY_DEPLOYMENTS, { context, namespaces }],
    queryFn: () => ListDeployments(effectiveNamespace),
    ...DEFAULT_QUERY_OPTIONS,
    enabled: !!context,
  });

  // Merge event-driven data locally: prefer event-filtered deployments over fetched data if available.
  // Filter cluster-wide event list by namespace membership.
  const mergedData = useMemo(() => {
    let baseData = query.data;
    if (latestDeployments.length) baseData = filterByNamespaces(latestDeployments, namespaces);
    return callback?.select ? callback.select(baseData) : baseData;
  }, [latestDeployments, query.data, namespaces, callback]);

  return {
    ...query,
    data: mergedData,
  };
};

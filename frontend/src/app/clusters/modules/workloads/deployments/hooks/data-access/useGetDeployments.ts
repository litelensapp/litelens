import { DEFAULT_QUERY_OPTIONS } from "../../../../../../shared/api/api";
import type { UseQueryCallback } from "@litelens/design-system";
import { useQuery } from "@tanstack/react-query";
import { useMemo } from "react";
import { QUERY_KEY_DEPLOYMENTS } from "../../api/api.const";
import type { Deployment } from "../../api/resources";
import { ListDeployments } from "../../api/resources";
import { useDeploymentsUpdateEvents } from "../async-events/useDeploymentsUpdateEvents";

export const useGetDeployments = (
  input: { context: string; namespace: string },
  callback?: UseQueryCallback<Deployment[]>
) => {
  const { context, namespace } = input;
  const latestDeployments = useDeploymentsUpdateEvents();

  const query = useQuery<Deployment[], Error>({
    queryKey: [QUERY_KEY_DEPLOYMENTS, { context, namespace }],
    queryFn: () => ListDeployments(namespace),
    ...DEFAULT_QUERY_OPTIONS,
    enabled: !!context,
  });

  // Merge event-driven data locally: prefer event-filtered deployments over fetched data if available.
  // Filter cluster-wide event list to this hook's namespace (or include all if namespace === "").
  const mergedData = useMemo(() => {
    let baseData = query.data;
    if (latestDeployments.length)
      baseData =
        namespace === ""
          ? latestDeployments
          : latestDeployments.filter((dep) => dep.Namespace === namespace);
    return callback?.select ? callback.select(baseData) : baseData;
  }, [latestDeployments, query.data, namespace, callback]);

  return {
    ...query,
    data: mergedData,
  };
};

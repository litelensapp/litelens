import { useQuery } from "@tanstack/react-query";
import { useMemo } from "react";
import { QUERY_KEY_DEPLOYMENT_DETAIL } from "../../api/api.const";
import { DEFAULT_QUERY_OPTIONS } from "../../../../../../shared/api/api";
import type { Deployment } from "../../api/resources";
import { GetDeploymentByName } from "../../api/resources";
import { useDeploymentsUpdateEvents } from "../async-events/useDeploymentsUpdateEvents";

export const useGetDeploymentDetail = (context: string, namespace: string, name: string) => {
  const latestDeployments = useDeploymentsUpdateEvents();

  const query = useQuery<Deployment, Error>({
    queryKey: [QUERY_KEY_DEPLOYMENT_DETAIL, { context, namespace, name }],
    queryFn: () => GetDeploymentByName(namespace, name),
    ...DEFAULT_QUERY_OPTIONS,
    enabled: !!context && !!namespace && !!name,
  });

  // Merge event-driven data: prefer matched deployment from latest event if available.
  const mergedData = useMemo(() => {
    const matchedDeployment = latestDeployments.find(
      (d) => d.Namespace === namespace && d.Name === name
    );
    if (matchedDeployment) return matchedDeployment;
    return query.data;
  }, [latestDeployments, query.data, namespace, name]);

  return {
    ...query,
    data: mergedData,
  };
};

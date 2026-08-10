import { useQuery, useQueryClient } from "@tanstack/react-query";
import { useEffect, useMemo } from "react";
import { QUERY_KEY_DEPLOYMENT_YAML } from "../../api/api.const";
import { DEFAULT_QUERY_OPTIONS } from "../../../../../../shared/api/api";
import { GetDeploymentYAML } from "../../api/resources";
import { useDeploymentsUpdateEvents } from "../async-events/useDeploymentsUpdateEvents";

export function useGetDeploymentYAML(
  context: string,
  namespace: string,
  name: string,
  enabled = true
) {
  const queryClient = useQueryClient();
  const latestDeployments = useDeploymentsUpdateEvents();

  const query = useQuery({
    queryKey: [QUERY_KEY_DEPLOYMENT_YAML, { context, namespace, name }],
    queryFn: () => GetDeploymentYAML(namespace, name),
    ...DEFAULT_QUERY_OPTIONS,
    enabled: !!context && !!namespace && !!name && enabled,
  });

  // Invalidate YAML cache for this deployment when a matching deployment update is received.
  // Use a stable derived value (serialized deployment key) as dependency to avoid invalidating
  // on every unrelated event churn.
  const deploymentKeyDependency = useMemo(() => {
    const matchedDeployment = latestDeployments.find(
      (d) => d.Namespace === namespace && d.Name === name
    );
    if (matchedDeployment) return JSON.stringify(matchedDeployment);
    return null;
  }, [latestDeployments, namespace, name]);

  useEffect(() => {
    if (deploymentKeyDependency)
      queryClient.invalidateQueries({
        queryKey: [QUERY_KEY_DEPLOYMENT_YAML, { context, namespace, name }],
      });
  }, [deploymentKeyDependency, context, namespace, name, queryClient]);

  return query;
}

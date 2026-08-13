import { useQuery } from "@tanstack/react-query";
import { DEFAULT_QUERY_OPTIONS } from "../../../../../../shared/api/api";
import { QUERY_KEY_DEPLOYMENTS } from "../../api/api.const";
import type { DeploymentSummary } from "../../api/resources";
import { GetDeploymentsSummary } from "../../api/resources";

export const useGetDeploymentsSummary = (input: { context: string; namespace: string }) => {
  const { context, namespace } = input;

  return useQuery<DeploymentSummary, Error>({
    queryKey: [QUERY_KEY_DEPLOYMENTS, "summary", { context, namespace }],
    queryFn: () => GetDeploymentsSummary(namespace),
    ...DEFAULT_QUERY_OPTIONS,
    enabled: !!context,
  });
};

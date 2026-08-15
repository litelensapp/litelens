import { useQuery } from "@tanstack/react-query";
import { DEFAULT_QUERY_OPTIONS } from "../../../../../../shared/api/api";
import { QUERY_KEY_DEPLOYMENTS } from "../../api/api.const";
import type { DeploymentSummary } from "../../api/resources";
import { GetDeploymentsSummary } from "../../api/resources";
import { getEffectiveNamespace } from "../../../../../shared/utils/namespaceFiltering";

export const useGetDeploymentsSummary = (input: { context: string; namespaces: string[] }) => {
  const { context, namespaces } = input;
  const effectiveNamespace = getEffectiveNamespace(namespaces);

  return useQuery<DeploymentSummary, Error>({
    queryKey: [QUERY_KEY_DEPLOYMENTS, "summary", { context, namespaces }],
    queryFn: () => GetDeploymentsSummary(effectiveNamespace),
    ...DEFAULT_QUERY_OPTIONS,
    enabled: !!context,
  });
};

import { useQuery } from "@tanstack/react-query";
import { DEFAULT_QUERY_OPTIONS } from "../../../../../../shared/api/api";
import { QUERY_KEY_PODS } from "../../api/api.const";
import type { PodSummary } from "../../api/resources";
import { GetPodsSummary } from "../../api/resources";
import { getEffectiveNamespace } from "../../../../../shared/utils/namespaceFiltering";

export const useGetPodsSummary = (input: { context: string; namespaces: string[] }) => {
  const { context, namespaces } = input;
  const effectiveNamespace = getEffectiveNamespace(namespaces);

  return useQuery<PodSummary, Error>({
    queryKey: [QUERY_KEY_PODS, "summary", { context, namespaces }],
    queryFn: () => GetPodsSummary(effectiveNamespace),
    ...DEFAULT_QUERY_OPTIONS,
    enabled: !!context,
  });
};

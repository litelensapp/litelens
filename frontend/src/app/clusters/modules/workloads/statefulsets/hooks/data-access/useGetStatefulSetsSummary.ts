import { useQuery } from "@tanstack/react-query";
import { DEFAULT_QUERY_OPTIONS } from "../../../../../../shared/api/api";
import { QUERY_KEY_STATEFULSETS } from "../../api/api.const";
import type { StatefulSetSummary } from "../../api/resources";
import { GetStatefulSetsSummary } from "../../api/resources";
import { getEffectiveNamespace } from "../../../../../shared/utils/namespaceFiltering";

export const useGetStatefulSetsSummary = (input: { context: string; namespaces: string[] }) => {
  const { context, namespaces } = input;
  const effectiveNamespace = getEffectiveNamespace(namespaces);

  return useQuery<StatefulSetSummary, Error>({
    queryKey: [QUERY_KEY_STATEFULSETS, "summary", { context, namespaces }],
    queryFn: () => GetStatefulSetsSummary(effectiveNamespace),
    ...DEFAULT_QUERY_OPTIONS,
    enabled: !!context,
  });
};

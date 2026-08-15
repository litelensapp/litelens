import { useQuery } from "@tanstack/react-query";
import { DEFAULT_QUERY_OPTIONS } from "../../../../../../shared/api/api";
import { QUERY_KEY_DAEMONSETS } from "../../api/api.const";
import type { DaemonSetSummary } from "../../api/resources";
import { GetDaemonSetsSummary } from "../../api/resources";
import { getEffectiveNamespace } from "../../../../../shared/utils/namespaceFiltering";

export const useGetDaemonSetsSummary = (input: { context: string; namespaces: string[] }) => {
  const { context, namespaces } = input;
  const effectiveNamespace = getEffectiveNamespace(namespaces);

  return useQuery<DaemonSetSummary, Error>({
    queryKey: [QUERY_KEY_DAEMONSETS, "summary", { context, namespaces }],
    queryFn: () => GetDaemonSetsSummary(effectiveNamespace),
    ...DEFAULT_QUERY_OPTIONS,
    enabled: !!context,
  });
};

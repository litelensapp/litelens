import { useQuery } from "@tanstack/react-query";
import { DEFAULT_QUERY_OPTIONS } from "../../../../../../shared/api/api";
import { QUERY_KEY_DAEMONSETS } from "../../api/api.const";
import type { DaemonSetSummary } from "../../api/resources";
import { GetDaemonSetsSummary } from "../../api/resources";

export const useGetDaemonSetsSummary = (input: { context: string; namespace: string }) => {
  const { context, namespace } = input;

  return useQuery<DaemonSetSummary, Error>({
    queryKey: [QUERY_KEY_DAEMONSETS, "summary", { context, namespace }],
    queryFn: () => GetDaemonSetsSummary(namespace),
    ...DEFAULT_QUERY_OPTIONS,
    enabled: !!context,
  });
};

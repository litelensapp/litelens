import { useQuery } from "@tanstack/react-query";
import { DEFAULT_QUERY_OPTIONS } from "../../../../../../shared/api/api";
import { QUERY_KEY_REPLICASETS } from "../../api/api.const";
import type { ReplicaSetSummary } from "../../api/resources";
import { GetReplicaSetsSummary } from "../../api/resources";
import { getEffectiveNamespace } from "../../../../../shared/utils/namespaceFiltering";

export const useGetReplicaSetsSummary = (input: { context: string; namespaces: string[] }) => {
  const { context, namespaces } = input;
  const effectiveNamespace = getEffectiveNamespace(namespaces);

  return useQuery<ReplicaSetSummary, Error>({
    queryKey: [QUERY_KEY_REPLICASETS, "summary", { context, namespaces }],
    queryFn: () => GetReplicaSetsSummary(effectiveNamespace),
    ...DEFAULT_QUERY_OPTIONS,
    enabled: !!context,
  });
};

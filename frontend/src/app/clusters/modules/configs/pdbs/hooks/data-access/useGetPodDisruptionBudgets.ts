import { DEFAULT_QUERY_OPTIONS } from "../../../../../../shared/api/api";
import type { UseQueryCallback } from "@litelens/design-system";
import { useMemo } from "react";
import { useQuery } from "@tanstack/react-query";
import { QUERY_KEY_PDBS } from "../../api/api.const";
import type { PodDisruptionBudget } from "../../api/resources";
import { ListPodDisruptionBudgets } from "../../api/resources";
import {
  getEffectiveNamespace,
  filterByNamespaces,
} from "../../../../../shared/utils/namespaceFiltering";
import { usePodDisruptionBudgetsUpdateEvents } from "../async-events/usePodDisruptionBudgetsUpdateEvents";

export const useGetPodDisruptionBudgets = (
  input: { context: string; namespaces: string[] },
  callback?: UseQueryCallback<PodDisruptionBudget[]>
) => {
  const { context, namespaces } = input;
  const effectiveNamespace = getEffectiveNamespace(namespaces);
  const latestPodDisruptionBudgets = usePodDisruptionBudgetsUpdateEvents();

  const query = useQuery<PodDisruptionBudget[], Error>({
    queryKey: [QUERY_KEY_PDBS, { context, namespaces }],
    queryFn: () => ListPodDisruptionBudgets(effectiveNamespace),
    ...DEFAULT_QUERY_OPTIONS,
    enabled: !!context,
  });

  const mergedData = useMemo(() => {
    let baseData = query.data;
    if (latestPodDisruptionBudgets.length)
      baseData = filterByNamespaces(latestPodDisruptionBudgets, namespaces);
    return callback?.select ? callback.select(baseData) : baseData;
  }, [latestPodDisruptionBudgets, query.data, namespaces, callback]);

  return { ...query, data: mergedData };
};

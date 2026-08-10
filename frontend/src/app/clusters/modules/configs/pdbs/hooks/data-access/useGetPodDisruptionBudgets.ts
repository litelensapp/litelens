import { DEFAULT_QUERY_OPTIONS } from "../../../../../../shared/api/api";
import type { UseQueryCallback } from "@litelens/design-system";
import { useQuery } from "@tanstack/react-query";
import { useMemo } from "react";
import { QUERY_KEY_PDBS } from "../../api/api.const";
import type { PodDisruptionBudget } from "../../api/resources";
import { ListPodDisruptionBudgets } from "../../api/resources";
import { usePodDisruptionBudgetsUpdateEvents } from "../async-events/usePodDisruptionBudgetsUpdateEvents";

export const useGetPodDisruptionBudgets = (
  input: { context: string; namespace: string },
  callback?: UseQueryCallback<PodDisruptionBudget[]>
) => {
  const { context, namespace } = input;
  const latestPodDisruptionBudgets = usePodDisruptionBudgetsUpdateEvents(namespace);

  const query = useQuery<PodDisruptionBudget[], Error>({
    queryKey: [QUERY_KEY_PDBS, { context, namespace }],
    queryFn: () => ListPodDisruptionBudgets(namespace),
    ...DEFAULT_QUERY_OPTIONS,
    enabled: !!context,
  });

  const mergedData = useMemo(() => {
    let baseData = query.data;
    if (latestPodDisruptionBudgets.length)
      baseData =
        namespace === ""
          ? latestPodDisruptionBudgets
          : latestPodDisruptionBudgets.filter((pdb) => pdb.Namespace === namespace);
    return callback?.select ? callback.select(baseData) : baseData;
  }, [latestPodDisruptionBudgets, query.data, namespace, callback]);

  return { ...query, data: mergedData };
};

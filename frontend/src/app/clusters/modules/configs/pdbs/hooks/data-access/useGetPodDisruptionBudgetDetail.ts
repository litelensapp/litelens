import { DEFAULT_QUERY_OPTIONS } from "../../../../../../shared/api/api";
import { useQuery, useQueryClient } from "@tanstack/react-query";
import { useEffect, useMemo } from "react";
import { QUERY_KEY_PDB_DETAIL } from "../../api/api.const";
import type { PodDisruptionBudgetDetail } from "../../api/resources";
import { GetPodDisruptionBudgetByName } from "../../api/resources";
import { usePodDisruptionBudgetsUpdateEvents } from "../async-events/usePodDisruptionBudgetsUpdateEvents";

export const useGetPodDisruptionBudgetDetail = (
  context: string,
  namespace: string,
  name: string
) => {
  const queryClient = useQueryClient();
  const latestPodDisruptionBudgets = usePodDisruptionBudgetsUpdateEvents(namespace);

  const query = useQuery<PodDisruptionBudgetDetail, Error>({
    queryKey: [QUERY_KEY_PDB_DETAIL, { context, namespace, name }],
    queryFn: () => GetPodDisruptionBudgetByName(namespace, name),
    ...DEFAULT_QUERY_OPTIONS,
    enabled: !!context && !!namespace && !!name,
  });

  const pdbKeyDependency = useMemo(() => {
    const matchedPodDisruptionBudget = latestPodDisruptionBudgets.find(
      (pdb) => pdb.Namespace === namespace && pdb.Name === name
    );
    return matchedPodDisruptionBudget ? JSON.stringify(matchedPodDisruptionBudget) : null;
  }, [latestPodDisruptionBudgets, namespace, name]);

  useEffect(() => {
    if (pdbKeyDependency)
      queryClient.invalidateQueries({
        queryKey: [QUERY_KEY_PDB_DETAIL, { context, namespace, name }],
      });
  }, [pdbKeyDependency, context, namespace, name, queryClient]);

  return query;
};

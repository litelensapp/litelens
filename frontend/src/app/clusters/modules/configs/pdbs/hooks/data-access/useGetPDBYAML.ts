import { DEFAULT_QUERY_OPTIONS } from "../../../../../../shared/api/api";
import { QUERY_KEY_PDB_YAML } from "../../api/api.const";
import { GetPDBYAML } from "../../api/resources";
import { useQuery, useQueryClient } from "@tanstack/react-query";
import { useEffect, useMemo } from "react";
import { usePodDisruptionBudgetsUpdateEvents } from "../async-events/usePodDisruptionBudgetsUpdateEvents";

export function useGetPDBYAML(context: string, namespace: string, name: string, enabled = true) {
  const queryClient = useQueryClient();
  const latestPodDisruptionBudgets = usePodDisruptionBudgetsUpdateEvents();

  const query = useQuery({
    queryKey: [QUERY_KEY_PDB_YAML, { context, namespace, name }],
    queryFn: () => GetPDBYAML(namespace, name),
    ...DEFAULT_QUERY_OPTIONS,
    enabled: !!context && !!namespace && !!name && enabled,
  });

  const pdbKeyDependency = useMemo(() => {
    const matchedPodDisruptionBudget = latestPodDisruptionBudgets.find(
      (pdb) => pdb.Namespace === namespace && pdb.Name === name
    );
    if (matchedPodDisruptionBudget) return JSON.stringify(matchedPodDisruptionBudget);
    return null;
  }, [latestPodDisruptionBudgets, namespace, name]);

  useEffect(() => {
    if (pdbKeyDependency)
      queryClient.invalidateQueries({
        queryKey: [QUERY_KEY_PDB_YAML, { context, namespace, name }],
      });
  }, [pdbKeyDependency, context, namespace, name, queryClient]);

  return query;
}

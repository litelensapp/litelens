import { DEFAULT_QUERY_OPTIONS } from "../../../../../../shared/api/api";
import type { UseQueryCallback } from "@litelens/design-system";
import { useQuery } from "@tanstack/react-query";
import { useMemo } from "react";
import { QUERY_KEY_STATEFULSETS } from "../../api/api.const";
import type { StatefulSet } from "../../api/resources";
import { ListStatefulSets } from "../../api/resources";
import { useStatefulSetsUpdateEvents } from "../async-events/useStatefulSetsUpdateEvents";

export const useGetStatefulSets = (
  input: { context: string; namespace: string },
  callback?: UseQueryCallback<StatefulSet[]>
) => {
  const { context, namespace } = input;
  const latestStatefulSets = useStatefulSetsUpdateEvents();

  const query = useQuery<StatefulSet[], Error>({
    queryKey: [QUERY_KEY_STATEFULSETS, { context, namespace }],
    queryFn: () => ListStatefulSets(namespace),
    ...DEFAULT_QUERY_OPTIONS,
    enabled: !!context,
  });

  // Merge event-driven data locally: prefer event-filtered statefulsets over fetched data if available.
  // Filter cluster-wide event list to this hook's namespace (or include all if namespace === "").
  const mergedData = useMemo(() => {
    let baseData = query.data;
    if (latestStatefulSets.length)
      baseData =
        namespace === ""
          ? latestStatefulSets
          : latestStatefulSets.filter((ss) => ss.Namespace === namespace);
    return callback?.select ? callback.select(baseData) : baseData;
  }, [latestStatefulSets, query.data, namespace, callback]);

  return {
    ...query,
    data: mergedData,
  };
};

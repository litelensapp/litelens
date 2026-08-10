import type { UseQueryCallback } from "@litelens/design-system";
import { useQuery } from "@tanstack/react-query";
import { useMemo } from "react";
import { DEFAULT_QUERY_OPTIONS } from "../../../../../../shared/api/api";
import { QUERY_KEY_PODS } from "../../api/api.const";
import type { Pod } from "../../api/resources";
import { ListPods } from "../../api/resources";
import { usePodsUpdateEvents } from "../async-events/usePodsUpdateEvents";

export const useGetPods = (
  input: { context: string; namespace: string },
  callback?: UseQueryCallback<Pod[]>
) => {
  const { context, namespace } = input;
  const latestPods = usePodsUpdateEvents(namespace);

  const query = useQuery<Pod[], Error>({
    queryKey: [QUERY_KEY_PODS, { context, namespace }],
    queryFn: () => ListPods(namespace),
    ...DEFAULT_QUERY_OPTIONS,
    enabled: !!context,
  });

  // Merge event-driven data locally: prefer event-filtered pods over fetched data if available.
  // Filter cluster-wide event list to this hook's namespace (or include all if namespace === "").
  const mergedData = useMemo(() => {
    let baseData = query.data;
    if (latestPods.length)
      baseData =
        namespace === "" ? latestPods : latestPods.filter((pod) => pod.Namespace === namespace);
    return callback?.select ? callback.select(baseData) : baseData;
  }, [latestPods, query.data, namespace, callback]);

  // Effective loading flag: if we've received at least one event-driven update,
  // we have real data even if query is re-fetching. Only show loading if query
  // is truly loading AND we haven't received any live event data yet.
  const isLoading = latestPods.length === 0 && query.isLoading;

  return {
    ...query,
    data: mergedData,
    isLoading,
  };
};

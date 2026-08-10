import { DEFAULT_QUERY_OPTIONS } from "../../../../../../shared/api/api";
import type { UseQueryCallback } from "@litelens/design-system";
import { useQuery } from "@tanstack/react-query";
import { useMemo } from "react";
import { QUERY_KEY_DAEMONSETS } from "../../api/api.const";
import type { DaemonSet } from "../../api/resources";
import { ListDaemonSets } from "../../api/resources";
import { useDaemonSetsUpdateEvents } from "../async-events/useDaemonSetsUpdateEvents";

export const useGetDaemonSets = (
  input: { context: string; namespace: string },
  callback?: UseQueryCallback<DaemonSet[]>
) => {
  const { context, namespace } = input;
  const latestDaemonSets = useDaemonSetsUpdateEvents();

  const query = useQuery<DaemonSet[], Error>({
    queryKey: [QUERY_KEY_DAEMONSETS, { context, namespace }],
    queryFn: () => ListDaemonSets(namespace),
    ...DEFAULT_QUERY_OPTIONS,
    enabled: !!context,
  });

  // Merge event-driven data locally: prefer event-filtered daemonsets over fetched data if available.
  // Filter cluster-wide event list to this hook's namespace (or include all if namespace === "").
  const mergedData = useMemo(() => {
    let baseData = query.data;
    if (latestDaemonSets.length)
      baseData =
        namespace === ""
          ? latestDaemonSets
          : latestDaemonSets.filter((ds) => ds.Namespace === namespace);
    return callback?.select ? callback.select(baseData) : baseData;
  }, [latestDaemonSets, query.data, namespace, callback]);

  return {
    ...query,
    data: mergedData,
  };
};

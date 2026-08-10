import { DEFAULT_QUERY_OPTIONS } from "../../../../../../shared/api/api";
import type { UseQueryCallback } from "@litelens/design-system";
import { useQuery } from "@tanstack/react-query";
import { useMemo } from "react";
import { QUERY_KEY_REPLICASETS } from "../../api/api.const";
import type { ReplicaSet } from "../../api/resources";
import { ListReplicaSets } from "../../api/resources";
import { useReplicaSetsUpdateEvents } from "../async-events/useReplicaSetsUpdateEvents";

export const useGetReplicaSets = (
  input: { context: string; namespace: string },
  callback?: UseQueryCallback<ReplicaSet[]>
) => {
  const { context, namespace } = input;
  const latestReplicaSets = useReplicaSetsUpdateEvents();

  const query = useQuery<ReplicaSet[], Error>({
    queryKey: [QUERY_KEY_REPLICASETS, { context, namespace }],
    queryFn: () => ListReplicaSets(namespace),
    ...DEFAULT_QUERY_OPTIONS,
    enabled: !!context,
  });

  // Merge event-driven data locally: prefer event-filtered replicasets over fetched data if available.
  // Filter cluster-wide event list to this hook's namespace (or include all if namespace === "").
  const mergedData = useMemo(() => {
    let baseData = query.data;
    if (latestReplicaSets.length)
      baseData =
        namespace === ""
          ? latestReplicaSets
          : latestReplicaSets.filter((rs) => rs.Namespace === namespace);
    return callback?.select ? callback.select(baseData) : baseData;
  }, [latestReplicaSets, query.data, namespace, callback]);

  return {
    ...query,
    data: mergedData,
  };
};

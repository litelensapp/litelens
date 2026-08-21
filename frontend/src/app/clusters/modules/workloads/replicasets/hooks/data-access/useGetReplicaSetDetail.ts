import { useQuery } from "@tanstack/react-query";
import { useMemo } from "react";
import { QUERY_KEY_REPLICASET_DETAIL } from "../../api/api.const";
import { DEFAULT_QUERY_OPTIONS } from "../../../../../../shared/api/api";
import type { ReplicaSet } from "../../api/resources";
import { GetReplicaSetByName } from "../../api/resources";
import { useReplicaSetsUpdateEvents } from "../async-events/useReplicaSetsUpdateEvents";

export const useGetReplicaSetDetail = (context: string, namespace: string, name: string) => {
  const latestReplicaSets = useReplicaSetsUpdateEvents([namespace]);

  const query = useQuery<ReplicaSet, Error>({
    queryKey: [QUERY_KEY_REPLICASET_DETAIL, { context, namespace, name }],
    queryFn: () => GetReplicaSetByName(namespace, name),
    ...DEFAULT_QUERY_OPTIONS,
    enabled: !!context && !!namespace && !!name,
  });

  // Merge event-driven data: prefer matched replicaset from latest event if available.
  const mergedData = useMemo(() => {
    const matchedReplicaSet = latestReplicaSets.find(
      (rs) => rs.Namespace === namespace && rs.Name === name
    );
    if (matchedReplicaSet) return matchedReplicaSet;
    return query.data;
  }, [latestReplicaSets, query.data, namespace, name]);

  return {
    ...query,
    data: mergedData,
  };
};

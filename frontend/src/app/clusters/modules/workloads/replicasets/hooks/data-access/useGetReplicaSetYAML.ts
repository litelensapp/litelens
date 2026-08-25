import { useQuery, useQueryClient } from "@tanstack/react-query";
import { useEffect, useMemo } from "react";
import { QUERY_KEY_REPLICASET_YAML } from "../../api/api.const";
import { DEFAULT_QUERY_OPTIONS } from "../../../../../../shared/api/api";
import { GetReplicaSetYAML } from "../../api/resources";
import { useReplicaSetsUpdateEvents } from "../async-events/useReplicaSetsUpdateEvents";

export function useGetReplicaSetYAML(
  context: string,
  namespace: string,
  name: string,
  enabled = true
) {
  const queryClient = useQueryClient();
  const latestReplicaSets = useReplicaSetsUpdateEvents();

  const query = useQuery({
    queryKey: [QUERY_KEY_REPLICASET_YAML, { context, namespace, name }],
    queryFn: () => GetReplicaSetYAML(namespace, name),
    ...DEFAULT_QUERY_OPTIONS,
    enabled: !!context && !!namespace && !!name && enabled,
  });

  // Invalidate YAML cache for this replicaset when a matching replicaset update is received.
  // Use a stable derived value (serialized replicaset key) as dependency to avoid invalidating
  // on every unrelated event churn.
  const replicasetKeyDependency = useMemo(() => {
    const matchedReplicaSet = latestReplicaSets.find(
      (rs) => rs.Namespace === namespace && rs.Name === name
    );
    if (matchedReplicaSet) return JSON.stringify(matchedReplicaSet);
    return null;
  }, [latestReplicaSets, namespace, name]);

  useEffect(() => {
    if (replicasetKeyDependency)
      queryClient.invalidateQueries({
        queryKey: [QUERY_KEY_REPLICASET_YAML, { context, namespace, name }],
      });
  }, [replicasetKeyDependency, context, namespace, name, queryClient]);

  return query;
}

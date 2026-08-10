import { useQuery, useQueryClient } from "@tanstack/react-query";
import { useEffect, useMemo } from "react";
import { QUERY_KEY_STATEFULSET_YAML } from "../../api/api.const";
import { DEFAULT_QUERY_OPTIONS } from "../../../../../../shared/api/api";
import { GetStatefulSetYAML } from "../../api/resources";
import { useStatefulSetsUpdateEvents } from "../async-events/useStatefulSetsUpdateEvents";

export function useGetStatefulSetYAML(
  context: string,
  namespace: string,
  name: string,
  enabled = true
) {
  const queryClient = useQueryClient();
  const latestStatefulSets = useStatefulSetsUpdateEvents();

  const query = useQuery({
    queryKey: [QUERY_KEY_STATEFULSET_YAML, { context, namespace, name }],
    queryFn: () => GetStatefulSetYAML(namespace, name),
    ...DEFAULT_QUERY_OPTIONS,
    enabled: !!context && !!namespace && !!name && enabled,
  });

  // Invalidate YAML cache for this statefulset when a matching statefulset update is received.
  // Use a stable derived value (serialized statefulset key) as dependency to avoid invalidating
  // on every unrelated event churn.
  const statefulsetKeyDependency = useMemo(() => {
    const matchedStatefulSet = latestStatefulSets.find(
      (ss) => ss.Namespace === namespace && ss.Name === name
    );
    if (matchedStatefulSet) return JSON.stringify(matchedStatefulSet);
    return null;
  }, [latestStatefulSets, namespace, name]);

  useEffect(() => {
    if (statefulsetKeyDependency)
      queryClient.invalidateQueries({
        queryKey: [QUERY_KEY_STATEFULSET_YAML, { context, namespace, name }],
      });
  }, [statefulsetKeyDependency, context, namespace, name, queryClient]);

  return query;
}

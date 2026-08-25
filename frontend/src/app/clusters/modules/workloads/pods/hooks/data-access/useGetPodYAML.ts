import { useQuery, useQueryClient } from "@tanstack/react-query";
import { useEffect, useMemo } from "react";
import { DEFAULT_QUERY_OPTIONS } from "../../../../../../shared/api/api";
import { QUERY_KEY_POD_YAML } from "../../api/api.const";
import { GetPodYAML } from "../../api/resources";
import { usePodsUpdateEvents } from "../async-events/usePodsUpdateEvents";

export function useGetPodYAML(context: string, namespace: string, name: string, enabled = true) {
  const queryClient = useQueryClient();
  // Live push-updates for this pod only arrive while its namespace is part of the active namespace filter.
  // Initial load via GetPodYAML is unaffected either way.
  const latestPods = usePodsUpdateEvents();

  const query = useQuery({
    queryKey: [QUERY_KEY_POD_YAML, { context, namespace, name }],
    queryFn: () => GetPodYAML(namespace, name),
    ...DEFAULT_QUERY_OPTIONS,
    enabled: !!context && !!namespace && !!name && enabled,
  });

  // Invalidate YAML cache for this pod when a matching pod update is received.
  // Use a stable derived value (serialized pod key) as dependency to avoid invalidating
  // on every unrelated event churn.
  const podKeyDependency = useMemo(() => {
    const matchedPod = latestPods.find((p) => p.Namespace === namespace && p.Name === name);
    // Serialize the pod to a stable string: changes only when the pod's content meaningfully changes.
    if (matchedPod) return JSON.stringify(matchedPod);
    return null;
  }, [latestPods, namespace, name]);

  useEffect(() => {
    if (podKeyDependency)
      queryClient.invalidateQueries({
        queryKey: [QUERY_KEY_POD_YAML, { context, namespace, name }],
      });
  }, [podKeyDependency, context, namespace, name, queryClient]);

  return query;
}

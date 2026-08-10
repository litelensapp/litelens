import { useQuery, useQueryClient } from "@tanstack/react-query";
import { useEffect, useMemo } from "react";
import { QUERY_KEY_DAEMONSET_YAML } from "../../api/api.const";
import { DEFAULT_QUERY_OPTIONS } from "../../../../../../shared/api/api";
import { GetDaemonSetYAML } from "../../api/resources";
import { useDaemonSetsUpdateEvents } from "../async-events/useDaemonSetsUpdateEvents";

export function useGetDaemonSetYAML(
  context: string,
  namespace: string,
  name: string,
  enabled = true
) {
  const queryClient = useQueryClient();
  const latestDaemonSets = useDaemonSetsUpdateEvents();

  const query = useQuery({
    queryKey: [QUERY_KEY_DAEMONSET_YAML, { context, namespace, name }],
    queryFn: () => GetDaemonSetYAML(namespace, name),
    ...DEFAULT_QUERY_OPTIONS,
    enabled: !!context && !!namespace && !!name && enabled,
  });

  // Invalidate YAML cache for this daemonset when a matching daemonset update is received.
  // Use a stable derived value (serialized daemonset key) as dependency to avoid invalidating
  // on every unrelated event churn.
  const daemonsetKeyDependency = useMemo(() => {
    const matchedDaemonSet = latestDaemonSets.find(
      (ds) => ds.Namespace === namespace && ds.Name === name
    );
    if (matchedDaemonSet) return JSON.stringify(matchedDaemonSet);
    return null;
  }, [latestDaemonSets, namespace, name]);

  useEffect(() => {
    if (daemonsetKeyDependency)
      queryClient.invalidateQueries({
        queryKey: [QUERY_KEY_DAEMONSET_YAML, { context, namespace, name }],
      });
  }, [daemonsetKeyDependency, context, namespace, name, queryClient]);

  return query;
}

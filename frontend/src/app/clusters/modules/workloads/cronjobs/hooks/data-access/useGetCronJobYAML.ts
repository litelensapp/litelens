import { useQuery, useQueryClient } from "@tanstack/react-query";
import { useEffect, useMemo } from "react";
import { QUERY_KEY_CRONJOB_YAML } from "../../api/api.const";
import { DEFAULT_QUERY_OPTIONS } from "../../../../../../shared/api/api";
import { GetCronJobYAML } from "../../api/resources";
import { useCronJobsUpdateEvents } from "../async-events/useCronJobsUpdateEvents";

export function useGetCronJobYAML(
  context: string,
  namespace: string,
  name: string,
  enabled = true
) {
  const queryClient = useQueryClient();
  const latestCronJobs = useCronJobsUpdateEvents([namespace]);

  const query = useQuery({
    queryKey: [QUERY_KEY_CRONJOB_YAML, { context, namespace, name }],
    queryFn: () => GetCronJobYAML(namespace, name),
    ...DEFAULT_QUERY_OPTIONS,
    enabled: !!context && !!namespace && !!name && enabled,
  });

  // Invalidate YAML cache for this cronjob when a matching cronjob update is received.
  // Use a stable derived value (serialized cronjob key) as dependency to avoid invalidating
  // on every unrelated event churn.
  const cronJobKeyDependency = useMemo(() => {
    const matchedCronJob = latestCronJobs.find(
      (cj) => cj.Namespace === namespace && cj.Name === name
    );
    // Serialize the cronjob to a stable string: changes only when the cronjob's content meaningfully changes.
    if (matchedCronJob) return JSON.stringify(matchedCronJob);
    return null;
  }, [latestCronJobs, namespace, name]);

  useEffect(() => {
    if (cronJobKeyDependency)
      queryClient.invalidateQueries({
        queryKey: [QUERY_KEY_CRONJOB_YAML, { context, namespace, name }],
      });
  }, [cronJobKeyDependency, context, namespace, name, queryClient]);

  return query;
}

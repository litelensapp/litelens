import { useQuery, useQueryClient } from "@tanstack/react-query";
import { useEffect, useMemo } from "react";
import { QUERY_KEY_JOB_YAML } from "../../api/api.const";
import { DEFAULT_QUERY_OPTIONS } from "../../../../../../shared/api/api";
import { GetJobYAML } from "../../api/resources";
import { useJobsUpdateEvents } from "../async-events/useJobsUpdateEvents";

export function useGetJobYAML(context: string, namespace: string, name: string, enabled = true) {
  const queryClient = useQueryClient();
  const latestJobs = useJobsUpdateEvents();

  const query = useQuery({
    queryKey: [QUERY_KEY_JOB_YAML, { context, namespace, name }],
    queryFn: () => GetJobYAML(namespace, name),
    ...DEFAULT_QUERY_OPTIONS,
    enabled: !!context && !!namespace && !!name && enabled,
  });

  // Invalidate YAML cache for this job when a matching job update is received.
  // Use a stable derived value (serialized job key) as dependency to avoid invalidating
  // on every unrelated event churn.
  const jobKeyDependency = useMemo(() => {
    const matchedJob = latestJobs.find((j) => j.Namespace === namespace && j.Name === name);
    // Serialize the job to a stable string: changes only when the job's content meaningfully changes.
    if (matchedJob) return JSON.stringify(matchedJob);
    return null;
  }, [latestJobs, namespace, name]);

  useEffect(() => {
    if (jobKeyDependency)
      queryClient.invalidateQueries({
        queryKey: [QUERY_KEY_JOB_YAML, { context, namespace, name }],
      });
  }, [jobKeyDependency, context, namespace, name, queryClient]);

  return query;
}

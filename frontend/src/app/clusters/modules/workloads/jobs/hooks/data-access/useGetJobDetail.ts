import { useQuery } from "@tanstack/react-query";
import { useMemo } from "react";
import { QUERY_KEY_JOB_DETAIL } from "../../api/api.const";
import { DEFAULT_QUERY_OPTIONS } from "../../../../../../shared/api/api";
import type { Job } from "../../api/resources";
import { GetJobByName } from "../../api/resources";
import { useJobsUpdateEvents } from "../async-events/useJobsUpdateEvents";

export const useGetJobDetail = (context: string, namespace: string, name: string) => {
  const latestJobs = useJobsUpdateEvents([namespace]);

  const query = useQuery<Job, Error>({
    queryKey: [QUERY_KEY_JOB_DETAIL, { context, namespace, name }],
    queryFn: () => GetJobByName(namespace, name),
    ...DEFAULT_QUERY_OPTIONS,
    enabled: !!context && !!namespace && !!name,
  });

  // Merge event-driven data: prefer matched job from latest event if available.
  const mergedData = useMemo(() => {
    const matchedJob = latestJobs.find((j) => j.Namespace === namespace && j.Name === name);
    if (matchedJob) return matchedJob;
    return query.data;
  }, [latestJobs, query.data, namespace, name]);

  return {
    ...query,
    data: mergedData,
  };
};

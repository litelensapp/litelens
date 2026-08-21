import { useQuery } from "@tanstack/react-query";
import { useMemo } from "react";
import { QUERY_KEY_CRONJOB_DETAIL } from "../../api/api.const";
import { DEFAULT_QUERY_OPTIONS } from "../../../../../../shared/api/api";
import type { CronJob } from "../../api/resources";
import { GetCronJobByName } from "../../api/resources";
import { useCronJobsUpdateEvents } from "../async-events/useCronJobsUpdateEvents";

export const useGetCronJobDetail = (context: string, namespace: string, name: string) => {
  const latestCronJobs = useCronJobsUpdateEvents([namespace]);

  const query = useQuery<CronJob, Error>({
    queryKey: [QUERY_KEY_CRONJOB_DETAIL, { context, namespace, name }],
    queryFn: () => GetCronJobByName(namespace, name),
    ...DEFAULT_QUERY_OPTIONS,
    enabled: !!context && !!namespace && !!name,
  });

  // Merge event-driven data: prefer matched cronjob from latest event if available.
  const mergedData = useMemo(() => {
    const matchedCronJob = latestCronJobs.find(
      (cj) => cj.Namespace === namespace && cj.Name === name
    );
    if (matchedCronJob) return matchedCronJob;
    return query.data;
  }, [latestCronJobs, query.data, namespace, name]);

  return {
    ...query,
    data: mergedData,
  };
};

import { DEFAULT_QUERY_OPTIONS } from "../../../../../../shared/api/api";
import type { UseQueryCallback } from "@litelens/design-system";
import { useQuery } from "@tanstack/react-query";
import { useMemo } from "react";
import { QUERY_KEY_CRONJOBS } from "../../api/api.const";
import type { CronJob } from "../../api/resources";
import { ListCronJobs } from "../../api/resources";
import { useCronJobsUpdateEvents } from "../async-events/useCronJobsUpdateEvents";

export const useGetCronJobs = (
  input: { context: string; namespace: string },
  callback?: UseQueryCallback<CronJob[]>
) => {
  const { context, namespace } = input;
  const latestCronJobs = useCronJobsUpdateEvents();

  const query = useQuery<CronJob[], Error>({
    queryKey: [QUERY_KEY_CRONJOBS, { context, namespace }],
    queryFn: () => ListCronJobs(namespace),
    ...DEFAULT_QUERY_OPTIONS,
    enabled: !!context,
  });

  // Merge event-driven data locally: prefer event-filtered cronjobs over fetched data if available.
  // Filter cluster-wide event list to this hook's namespace (or include all if namespace === "").
  const mergedData = useMemo(() => {
    let baseData = query.data;
    if (latestCronJobs.length)
      baseData =
        namespace === ""
          ? latestCronJobs
          : latestCronJobs.filter((cj) => cj.Namespace === namespace);
    return callback?.select ? callback.select(baseData) : baseData;
  }, [latestCronJobs, query.data, namespace, callback]);

  return {
    ...query,
    data: mergedData,
  };
};

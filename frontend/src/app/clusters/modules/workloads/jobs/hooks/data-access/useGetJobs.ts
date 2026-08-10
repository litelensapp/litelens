import { DEFAULT_QUERY_OPTIONS } from "../../../../../../shared/api/api";
import type { UseQueryCallback } from "@litelens/design-system";
import { useQuery } from "@tanstack/react-query";
import { useMemo } from "react";
import { QUERY_KEY_JOBS } from "../../api/api.const";
import type { Job } from "../../api/resources";
import { ListJobs } from "../../api/resources";
import { useJobsUpdateEvents } from "../async-events/useJobsUpdateEvents";

export const useGetJobs = (
  input: { context: string; namespace: string },
  callback?: UseQueryCallback<Job[]>
) => {
  const { context, namespace } = input;
  const latestJobs = useJobsUpdateEvents(namespace);

  const query = useQuery<Job[], Error>({
    queryKey: [QUERY_KEY_JOBS, { context, namespace }],
    queryFn: () => ListJobs(namespace),
    ...DEFAULT_QUERY_OPTIONS,
    enabled: !!context,
  });

  // Merge event-driven data locally: prefer event-filtered jobs over fetched data if available.
  // Filter cluster-wide event list to this hook's namespace (or include all if namespace === "").
  const mergedData = useMemo(() => {
    let baseData = query.data;
    if (latestJobs.length)
      baseData =
        namespace === "" ? latestJobs : latestJobs.filter((job) => job.Namespace === namespace);
    return callback?.select ? callback.select(baseData) : baseData;
  }, [latestJobs, query.data, namespace, callback]);

  return {
    ...query,
    data: mergedData,
  };
};

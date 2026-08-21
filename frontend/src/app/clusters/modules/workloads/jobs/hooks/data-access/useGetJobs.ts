import { DEFAULT_QUERY_OPTIONS } from "../../../../../../shared/api/api";
import type { UseQueryCallback } from "@litelens/core";
import { useMemo } from "react";
import { useQuery } from "@tanstack/react-query";
import { QUERY_KEY_JOBS } from "../../api/api.const";
import type { Job } from "../../api/resources";
import { ListJobs } from "../../api/resources";
import { filterByNamespaces } from "../../../../../shared/utils/namespaceFiltering";
import { useJobsUpdateEvents } from "../async-events/useJobsUpdateEvents";

export const useGetJobs = (
  input: { context: string; namespaces: string[] },
  callback?: UseQueryCallback<Job[]>
) => {
  const { context, namespaces } = input;
  const latestJobs = useJobsUpdateEvents(namespaces);

  const query = useQuery<Job[], Error>({
    queryKey: [QUERY_KEY_JOBS, { context, namespaces }],
    queryFn: () => ListJobs(namespaces),
    ...DEFAULT_QUERY_OPTIONS,
    enabled: !!context,
  });

  const mergedData = useMemo(() => {
    let baseData = query.data;
    if (latestJobs.length) baseData = filterByNamespaces(latestJobs, namespaces);
    return callback?.select ? callback.select(baseData) : baseData;
  }, [latestJobs, query.data, namespaces, callback]);

  return { ...query, data: mergedData };
};

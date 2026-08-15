import { DEFAULT_QUERY_OPTIONS } from "../../../../../../shared/api/api";
import type { UseQueryCallback } from "@litelens/design-system";
import { useMemo } from "react";
import { useQuery } from "@tanstack/react-query";
import { QUERY_KEY_CRONJOBS } from "../../api/api.const";
import type { CronJob } from "../../api/resources";
import { ListCronJobs } from "../../api/resources";
import {
  getEffectiveNamespace,
  filterByNamespaces,
} from "../../../../../shared/utils/namespaceFiltering";
import { useCronJobsUpdateEvents } from "../async-events/useCronJobsUpdateEvents";

export const useGetCronJobs = (
  input: { context: string; namespaces: string[] },
  callback?: UseQueryCallback<CronJob[]>
) => {
  const { context, namespaces } = input;
  const effectiveNamespace = getEffectiveNamespace(namespaces);
  const latestCronJobs = useCronJobsUpdateEvents();

  const query = useQuery<CronJob[], Error>({
    queryKey: [QUERY_KEY_CRONJOBS, { context, namespaces }],
    queryFn: () => ListCronJobs(effectiveNamespace),
    ...DEFAULT_QUERY_OPTIONS,
    enabled: !!context,
  });

  const mergedData = useMemo(() => {
    let baseData = query.data;
    if (latestCronJobs.length) baseData = filterByNamespaces(latestCronJobs, namespaces);
    return callback?.select ? callback.select(baseData) : baseData;
  }, [latestCronJobs, query.data, namespaces, callback]);

  return { ...query, data: mergedData };
};

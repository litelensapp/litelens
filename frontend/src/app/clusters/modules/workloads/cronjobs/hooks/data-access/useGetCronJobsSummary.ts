import { useQuery } from "@tanstack/react-query";
import { DEFAULT_QUERY_OPTIONS } from "../../../../../../shared/api/api";
import { QUERY_KEY_CRONJOBS } from "../../api/api.const";
import type { CronJobSummary } from "../../api/resources";
import { GetCronJobsSummary } from "../../api/resources";

export const useGetCronJobsSummary = (input: { context: string; namespace: string }) => {
  const { context, namespace } = input;

  return useQuery<CronJobSummary, Error>({
    queryKey: [QUERY_KEY_CRONJOBS, "summary", { context, namespace }],
    queryFn: () => GetCronJobsSummary(namespace),
    ...DEFAULT_QUERY_OPTIONS,
    enabled: !!context,
  });
};

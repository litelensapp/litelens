import { useQuery } from "@tanstack/react-query";
import { DEFAULT_QUERY_OPTIONS } from "../../../../../../shared/api/api";
import { QUERY_KEY_CRONJOBS } from "../../api/api.const";
import type { CronJobSummary } from "../../api/resources";
import { GetCronJobsSummary } from "../../api/resources";
import { getEffectiveNamespace } from "../../../../../shared/utils/namespaceFiltering";

export const useGetCronJobsSummary = (input: { context: string; namespaces: string[] }) => {
  const { context, namespaces } = input;
  const effectiveNamespace = getEffectiveNamespace(namespaces);

  return useQuery<CronJobSummary, Error>({
    queryKey: [QUERY_KEY_CRONJOBS, "summary", { context, namespaces }],
    queryFn: () => GetCronJobsSummary(effectiveNamespace),
    ...DEFAULT_QUERY_OPTIONS,
    enabled: !!context,
  });
};

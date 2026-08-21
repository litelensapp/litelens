import { DEFAULT_QUERY_OPTIONS } from "../../../../../../shared/api/api";
import type { UseQueryCallback } from "@litelens/core";
import { useMemo } from "react";
import { useQuery } from "@tanstack/react-query";
import { QUERY_KEY_LIMIT_RANGES } from "../../api/api.const";
import type { LimitRange } from "../../api/resources";
import { ListLimitRanges } from "../../api/resources";
import { filterByNamespaces } from "../../../../../shared/utils/namespaceFiltering";
import { useLimitRangesUpdateEvents } from "../async-events/useLimitRangesUpdateEvents";

export const useGetLimitRanges = (
  input: { context: string; namespaces: string[] },
  callback?: UseQueryCallback<LimitRange[]>
) => {
  const { context, namespaces } = input;
  const latestLimitRanges = useLimitRangesUpdateEvents(namespaces);

  const query = useQuery<LimitRange[], Error>({
    queryKey: [QUERY_KEY_LIMIT_RANGES, { context, namespaces }],
    queryFn: () => ListLimitRanges(namespaces),
    ...DEFAULT_QUERY_OPTIONS,
    enabled: !!context,
  });

  const mergedData = useMemo(() => {
    let baseData = query.data;
    if (latestLimitRanges.length) baseData = filterByNamespaces(latestLimitRanges, namespaces);
    return callback?.select ? callback.select(baseData) : baseData;
  }, [latestLimitRanges, query.data, namespaces, callback]);

  return { ...query, data: mergedData };
};

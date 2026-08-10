import { DEFAULT_QUERY_OPTIONS } from "../../../../../../shared/api/api";
import type { UseQueryCallback } from "@litelens/design-system";
import { useQuery } from "@tanstack/react-query";
import { useMemo } from "react";
import { QUERY_KEY_LIMIT_RANGES } from "../../api/api.const";
import type { LimitRange } from "../../api/resources";
import { ListLimitRanges } from "../../api/resources";
import { useLimitRangesUpdateEvents } from "../async-events/useLimitRangesUpdateEvents";

export const useGetLimitRanges = (
  input: { context: string; namespace: string },
  callback?: UseQueryCallback<LimitRange[]>
) => {
  const { context, namespace } = input;
  const latestLimitRanges = useLimitRangesUpdateEvents(namespace);

  const query = useQuery<LimitRange[], Error>({
    queryKey: [QUERY_KEY_LIMIT_RANGES, { context, namespace }],
    queryFn: () => ListLimitRanges(namespace),
    ...DEFAULT_QUERY_OPTIONS,
    enabled: !!context,
  });

  const mergedData = useMemo(() => {
    let baseData = query.data;
    if (latestLimitRanges.length)
      baseData =
        namespace === ""
          ? latestLimitRanges
          : latestLimitRanges.filter((lr) => lr.Namespace === namespace);
    return callback?.select ? callback.select(baseData) : baseData;
  }, [latestLimitRanges, query.data, namespace, callback]);

  return { ...query, data: mergedData };
};

import { DEFAULT_QUERY_OPTIONS } from "../../../../../../shared/api/api";
import { useQuery, useQueryClient } from "@tanstack/react-query";
import { useEffect, useMemo } from "react";
import { QUERY_KEY_LIMIT_RANGE_DETAIL } from "../../api/api.const";
import type { LimitRangeDetail } from "../../api/resources";
import { GetLimitRangeByName } from "../../api/resources";
import { useLimitRangesUpdateEvents } from "../async-events/useLimitRangesUpdateEvents";

export const useGetLimitRangeDetail = (context: string, namespace: string, name: string) => {
  const queryClient = useQueryClient();
  const latestLimitRanges = useLimitRangesUpdateEvents(namespace);

  const query = useQuery<LimitRangeDetail, Error>({
    queryKey: [QUERY_KEY_LIMIT_RANGE_DETAIL, { context, namespace, name }],
    queryFn: () => GetLimitRangeByName(namespace, name),
    ...DEFAULT_QUERY_OPTIONS,
    enabled: !!context && !!namespace && !!name,
  });

  const limitRangeKeyDependency = useMemo(() => {
    const matchedLimitRange = latestLimitRanges.find(
      (lr) => lr.Namespace === namespace && lr.Name === name
    );
    return matchedLimitRange ? JSON.stringify(matchedLimitRange) : null;
  }, [latestLimitRanges, namespace, name]);

  useEffect(() => {
    if (limitRangeKeyDependency)
      queryClient.invalidateQueries({
        queryKey: [QUERY_KEY_LIMIT_RANGE_DETAIL, { context, namespace, name }],
      });
  }, [limitRangeKeyDependency, context, namespace, name, queryClient]);

  return query;
};

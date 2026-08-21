import { DEFAULT_QUERY_OPTIONS } from "../../../../../../shared/api/api";
import { QUERY_KEY_LIMITRANGE_YAML } from "../../api/api.const";
import { GetLimitRangeYAML } from "../../api/resources";
import { useQuery, useQueryClient } from "@tanstack/react-query";
import { useEffect, useMemo } from "react";
import { useLimitRangesUpdateEvents } from "../async-events/useLimitRangesUpdateEvents";

export function useGetLimitRangeYAML(
  context: string,
  namespace: string,
  name: string,
  enabled = true
) {
  const queryClient = useQueryClient();
  const latestLimitRanges = useLimitRangesUpdateEvents([namespace]);

  const query = useQuery({
    queryKey: [QUERY_KEY_LIMITRANGE_YAML, { context, namespace, name }],
    queryFn: () => GetLimitRangeYAML(namespace, name),
    ...DEFAULT_QUERY_OPTIONS,
    enabled: !!context && !!namespace && !!name && enabled,
  });

  const lrKeyDependency = useMemo(() => {
    const matchedLimitRange = latestLimitRanges.find(
      (lr) => lr.Namespace === namespace && lr.Name === name
    );
    if (matchedLimitRange) return JSON.stringify(matchedLimitRange);
    return null;
  }, [latestLimitRanges, namespace, name]);

  useEffect(() => {
    if (lrKeyDependency)
      queryClient.invalidateQueries({
        queryKey: [QUERY_KEY_LIMITRANGE_YAML, { context, namespace, name }],
      });
  }, [lrKeyDependency, context, namespace, name, queryClient]);

  return query;
}

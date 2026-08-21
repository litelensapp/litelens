import { useQuery } from "@tanstack/react-query";
import { useMemo } from "react";
import { QUERY_KEY_STATEFULSET_DETAIL } from "../../api/api.const";
import { DEFAULT_QUERY_OPTIONS } from "../../../../../../shared/api/api";
import type { StatefulSet } from "../../api/resources";
import { GetStatefulSetByName } from "../../api/resources";
import { useStatefulSetsUpdateEvents } from "../async-events/useStatefulSetsUpdateEvents";

export const useGetStatefulSetDetail = (context: string, namespace: string, name: string) => {
  const latestStatefulSets = useStatefulSetsUpdateEvents([namespace]);

  const query = useQuery<StatefulSet, Error>({
    queryKey: [QUERY_KEY_STATEFULSET_DETAIL, { context, namespace, name }],
    queryFn: () => GetStatefulSetByName(namespace, name),
    ...DEFAULT_QUERY_OPTIONS,
    enabled: !!context && !!namespace && !!name,
  });

  // Merge event-driven data: prefer matched statefulset from latest event if available.
  const mergedData = useMemo(() => {
    const matchedStatefulSet = latestStatefulSets.find(
      (ss) => ss.Namespace === namespace && ss.Name === name
    );
    if (matchedStatefulSet) return matchedStatefulSet;
    return query.data;
  }, [latestStatefulSets, query.data, namespace, name]);

  return {
    ...query,
    data: mergedData,
  };
};

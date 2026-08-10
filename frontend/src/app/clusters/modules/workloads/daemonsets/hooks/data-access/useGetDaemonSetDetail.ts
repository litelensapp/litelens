import { useQuery } from "@tanstack/react-query";
import { useMemo } from "react";
import { QUERY_KEY_DAEMONSET_DETAIL } from "../../api/api.const";
import { DEFAULT_QUERY_OPTIONS } from "../../../../../../shared/api/api";
import type { DaemonSet } from "../../api/resources";
import { GetDaemonSetByName } from "../../api/resources";
import { useDaemonSetsUpdateEvents } from "../async-events/useDaemonSetsUpdateEvents";

export const useGetDaemonSetDetail = (context: string, namespace: string, name: string) => {
  const latestDaemonSets = useDaemonSetsUpdateEvents();

  const query = useQuery<DaemonSet, Error>({
    queryKey: [QUERY_KEY_DAEMONSET_DETAIL, { context, namespace, name }],
    queryFn: () => GetDaemonSetByName(namespace, name),
    ...DEFAULT_QUERY_OPTIONS,
    enabled: !!context && !!namespace && !!name,
  });

  // Merge event-driven data: prefer matched daemonset from latest event if available.
  const mergedData = useMemo(() => {
    const matchedDaemonSet = latestDaemonSets.find(
      (ds) => ds.Namespace === namespace && ds.Name === name
    );
    if (matchedDaemonSet) return matchedDaemonSet;
    return query.data;
  }, [latestDaemonSets, query.data, namespace, name]);

  return {
    ...query,
    data: mergedData,
  };
};

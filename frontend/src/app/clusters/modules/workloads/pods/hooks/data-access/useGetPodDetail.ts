import { useQuery } from "@tanstack/react-query";
import { useMemo } from "react";
import { DEFAULT_QUERY_OPTIONS } from "../../../../../../shared/api/api";
import { QUERY_KEY_POD_DETAIL } from "../../api/api.const";
import type { Pod } from "../../api/resources";
import { GetPodByName } from "../../api/resources";
import { usePodsUpdateEvents } from "../async-events/usePodsUpdateEvents";

export const useGetPodDetail = (context: string, namespace: string, name: string) => {
  // Live push-updates for this pod only arrive while its namespace is part of the active namespace filter.
  // Initial load via GetPodByName is unaffected either way.
  const latestPods = usePodsUpdateEvents();

  const query = useQuery<Pod, Error>({
    queryKey: [QUERY_KEY_POD_DETAIL, { context, namespace, name }],
    queryFn: () => GetPodByName(namespace, name),
    ...DEFAULT_QUERY_OPTIONS,
    enabled: !!context && !!namespace && !!name,
  });

  // Merge event-driven data: prefer matched pod from latest event if available.
  const mergedData = useMemo(() => {
    const matchedPod = latestPods.find((p) => p.Namespace === namespace && p.Name === name);
    if (matchedPod) return matchedPod;
    return query.data;
  }, [latestPods, query.data, namespace, name]);

  return {
    ...query,
    data: mergedData,
  };
};

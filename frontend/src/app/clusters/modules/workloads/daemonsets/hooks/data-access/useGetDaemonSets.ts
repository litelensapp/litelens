import { DEFAULT_QUERY_OPTIONS } from "../../../../../../shared/api/api";
import type { UseQueryCallback } from "@litelens/core";
import { useMemo } from "react";
import { useQuery } from "@tanstack/react-query";
import { QUERY_KEY_DAEMONSETS } from "../../api/api.const";
import type { DaemonSet } from "../../api/resources";
import { ListDaemonSets } from "../../api/resources";
import { filterByNamespaces } from "../../../../../shared/utils/namespaceFiltering";
import { useDaemonSetsUpdateEvents } from "../async-events/useDaemonSetsUpdateEvents";

export const useGetDaemonSets = (
  input: { context: string; namespaces: string[] },
  callback?: UseQueryCallback<DaemonSet[]>
) => {
  const { context, namespaces } = input;
  const latestDaemonSets = useDaemonSetsUpdateEvents(namespaces);

  const query = useQuery<DaemonSet[], Error>({
    queryKey: [QUERY_KEY_DAEMONSETS, { context, namespaces }],
    queryFn: () => ListDaemonSets(namespaces),
    ...DEFAULT_QUERY_OPTIONS,
    enabled: !!context,
  });

  const mergedData = useMemo(() => {
    let baseData = query.data;
    if (latestDaemonSets.length) baseData = filterByNamespaces(latestDaemonSets, namespaces);
    return callback?.select ? callback.select(baseData) : baseData;
  }, [latestDaemonSets, query.data, namespaces, callback]);

  return { ...query, data: mergedData };
};

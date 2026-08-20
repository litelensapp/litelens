import { DEFAULT_QUERY_OPTIONS } from "../../../../../../shared/api/api";
import type { UseQueryCallback } from "@litelens/core";
import { useMemo } from "react";
import { useQuery } from "@tanstack/react-query";
import { QUERY_KEY_STATEFULSETS } from "../../api/api.const";
import type { StatefulSet } from "../../api/resources";
import { ListStatefulSets } from "../../api/resources";
import {
  getEffectiveNamespace,
  filterByNamespaces,
} from "../../../../../shared/utils/namespaceFiltering";
import { useStatefulSetsUpdateEvents } from "../async-events/useStatefulSetsUpdateEvents";

export const useGetStatefulSets = (
  input: { context: string; namespaces: string[] },
  callback?: UseQueryCallback<StatefulSet[]>
) => {
  const { context, namespaces } = input;
  const effectiveNamespace = getEffectiveNamespace(namespaces);
  const latestStatefulSets = useStatefulSetsUpdateEvents();

  const query = useQuery<StatefulSet[], Error>({
    queryKey: [QUERY_KEY_STATEFULSETS, { context, namespaces }],
    queryFn: () => ListStatefulSets(effectiveNamespace),
    ...DEFAULT_QUERY_OPTIONS,
    enabled: !!context,
  });

  const mergedData = useMemo(() => {
    let baseData = query.data;
    if (latestStatefulSets.length) baseData = filterByNamespaces(latestStatefulSets, namespaces);
    return callback?.select ? callback.select(baseData) : baseData;
  }, [latestStatefulSets, query.data, namespaces, callback]);

  return { ...query, data: mergedData };
};

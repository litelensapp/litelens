import { DEFAULT_QUERY_OPTIONS } from "../../../../../../shared/api/api";
import type { UseQueryCallback } from "@litelens/core";
import { useMemo } from "react";
import { useQuery } from "@tanstack/react-query";
import { QUERY_KEY_REPLICASETS } from "../../api/api.const";
import type { ReplicaSet } from "../../api/resources";
import { ListReplicaSets } from "../../api/resources";
import {
  getEffectiveNamespace,
  filterByNamespaces,
} from "../../../../../shared/utils/namespaceFiltering";
import { useReplicaSetsUpdateEvents } from "../async-events/useReplicaSetsUpdateEvents";

export const useGetReplicaSets = (
  input: { context: string; namespaces: string[] },
  callback?: UseQueryCallback<ReplicaSet[]>
) => {
  const { context, namespaces } = input;
  const effectiveNamespace = getEffectiveNamespace(namespaces);
  const latestReplicaSets = useReplicaSetsUpdateEvents();

  const query = useQuery<ReplicaSet[], Error>({
    queryKey: [QUERY_KEY_REPLICASETS, { context, namespaces }],
    queryFn: () => ListReplicaSets(effectiveNamespace),
    ...DEFAULT_QUERY_OPTIONS,
    enabled: !!context,
  });

  const mergedData = useMemo(() => {
    let baseData = query.data;
    if (latestReplicaSets.length) baseData = filterByNamespaces(latestReplicaSets, namespaces);
    return callback?.select ? callback.select(baseData) : baseData;
  }, [latestReplicaSets, query.data, namespaces, callback]);

  return { ...query, data: mergedData };
};

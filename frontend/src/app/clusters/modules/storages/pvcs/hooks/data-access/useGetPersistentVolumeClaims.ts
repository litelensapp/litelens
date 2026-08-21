import { DEFAULT_QUERY_OPTIONS } from "../../../../../../shared/api/api";
import type { UseQueryCallback } from "@litelens/core";
import { useMemo } from "react";
import { useQuery } from "@tanstack/react-query";
import { QUERY_KEY_PVCS } from "../../api/api.const";
import type { PersistentVolumeClaim } from "../../api/resources";
import { ListPersistentVolumeClaims } from "../../api/resources";
import { filterByNamespaces } from "../../../../../shared/utils/namespaceFiltering";
import { usePersistentVolumeClaimsUpdateEvents } from "../async-events/usePersistentVolumeClaimsUpdateEvents";

export const useGetPersistentVolumeClaims = (
  input: { context: string; namespaces: string[] },
  callback?: UseQueryCallback<PersistentVolumeClaim[]>
) => {
  const { context, namespaces } = input;
  const latestPersistentVolumeClaims = usePersistentVolumeClaimsUpdateEvents(namespaces);

  const query = useQuery<PersistentVolumeClaim[], Error>({
    queryKey: [QUERY_KEY_PVCS, { context, namespaces }],
    queryFn: () => ListPersistentVolumeClaims(namespaces),
    ...DEFAULT_QUERY_OPTIONS,
    enabled: !!context,
  });

  const mergedData = useMemo(() => {
    let baseData = query.data;
    if (latestPersistentVolumeClaims.length)
      baseData = filterByNamespaces(latestPersistentVolumeClaims, namespaces);
    return callback?.select ? callback.select(baseData) : baseData;
  }, [latestPersistentVolumeClaims, query.data, namespaces, callback]);

  return { ...query, data: mergedData };
};

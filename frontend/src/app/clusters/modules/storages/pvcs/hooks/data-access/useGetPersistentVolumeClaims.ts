import { DEFAULT_QUERY_OPTIONS } from "../../../../../../shared/api/api";
import type { UseQueryCallback } from "@litelens/design-system";
import { useMemo } from "react";
import { useQuery } from "@tanstack/react-query";
import { QUERY_KEY_PVCS } from "../../api/api.const";
import type { PersistentVolumeClaim } from "../../api/resources";
import { ListPersistentVolumeClaims } from "../../api/resources";
import { usePersistentVolumeClaimsUpdateEvents } from "../async-events/usePersistentVolumeClaimsUpdateEvents";

export const useGetPersistentVolumeClaims = (
  input: { context: string; namespace: string },
  callback?: UseQueryCallback<PersistentVolumeClaim[]>
) => {
  const { context, namespace } = input;
  const latestPVCs = usePersistentVolumeClaimsUpdateEvents(namespace);
  const query = useQuery<PersistentVolumeClaim[], Error>({
    queryKey: [QUERY_KEY_PVCS, { context, namespace }],
    queryFn: () => ListPersistentVolumeClaims(namespace),
    ...DEFAULT_QUERY_OPTIONS,
    enabled: !!context,
  });

  const mergedData = useMemo(() => {
    let baseData = query.data;
    if (latestPVCs.length)
      baseData =
        namespace === "" ? latestPVCs : latestPVCs.filter((pvc) => pvc.Namespace === namespace);
    return callback?.select ? callback.select(baseData) : baseData;
  }, [latestPVCs, query.data, namespace, callback]);

  return { ...query, data: mergedData };
};

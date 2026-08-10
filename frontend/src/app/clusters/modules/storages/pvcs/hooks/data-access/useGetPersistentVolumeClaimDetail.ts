import { useEffect, useMemo } from "react";
import { useQuery, useQueryClient } from "@tanstack/react-query";
import { QUERY_KEY_PVC_DETAIL } from "../../api/api.const";
import { DEFAULT_QUERY_OPTIONS } from "../../../../../../shared/api/api";
import type { PersistentVolumeClaimDetail } from "../../api/resources";
import { GetPersistentVolumeClaimByName } from "../../api/resources";
import { usePersistentVolumeClaimsUpdateEvents } from "../async-events/usePersistentVolumeClaimsUpdateEvents";

export const useGetPersistentVolumeClaimDetail = (
  context: string,
  namespace: string,
  name: string
) => {
  const queryClient = useQueryClient();
  const latestPVCs = usePersistentVolumeClaimsUpdateEvents(namespace);
  const query = useQuery<PersistentVolumeClaimDetail, Error>({
    queryKey: [QUERY_KEY_PVC_DETAIL, { context, namespace, name }],
    queryFn: () => GetPersistentVolumeClaimByName(namespace, name),
    ...DEFAULT_QUERY_OPTIONS,
    enabled: !!context && !!namespace && !!name,
  });

  const matchedPVC = useMemo(
    () => latestPVCs.find((pvc) => pvc.Namespace === namespace && pvc.Name === name),
    [latestPVCs, namespace, name]
  );
  const matchedPVCKey = JSON.stringify(matchedPVC);

  useEffect(() => {
    if (matchedPVC) {
      queryClient.invalidateQueries({
        queryKey: [QUERY_KEY_PVC_DETAIL, { context, namespace, name }],
      });
    }
  }, [matchedPVC, matchedPVCKey, context, namespace, name, queryClient]);

  return query;
};

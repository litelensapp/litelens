import { useEffect, useMemo } from "react";
import { useQuery, useQueryClient } from "@tanstack/react-query";
import { QUERY_KEY_PERSISTENTVOLUMECLAIM_YAML } from "../../api/api.const";
import { DEFAULT_QUERY_OPTIONS } from "../../../../../../shared/api/api";
import { GetPersistentVolumeClaimYAML } from "../../api/resources";
import { usePersistentVolumeClaimsUpdateEvents } from "../async-events/usePersistentVolumeClaimsUpdateEvents";

export function useGetPersistentVolumeClaimYAML(
  context: string,
  namespace: string,
  name: string,
  enabled = true
) {
  const queryClient = useQueryClient();
  const latestPVCs = usePersistentVolumeClaimsUpdateEvents();
  const query = useQuery({
    queryKey: [QUERY_KEY_PERSISTENTVOLUMECLAIM_YAML, { context, namespace, name }],
    queryFn: () => GetPersistentVolumeClaimYAML(namespace, name),
    ...DEFAULT_QUERY_OPTIONS,
    enabled: !!context && !!namespace && !!name && enabled,
  });

  const matchedPVC = useMemo(
    () => latestPVCs.find((pvc) => pvc.Namespace === namespace && pvc.Name === name),
    [latestPVCs, namespace, name]
  );
  const matchedPVCKey = JSON.stringify(matchedPVC);

  useEffect(() => {
    if (matchedPVC) {
      queryClient.invalidateQueries({
        queryKey: [QUERY_KEY_PERSISTENTVOLUMECLAIM_YAML, { context, namespace, name }],
      });
    }
  }, [matchedPVC, matchedPVCKey, context, namespace, name, queryClient]);

  return query;
}

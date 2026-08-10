import { DEFAULT_QUERY_OPTIONS } from "../../../../../../shared/api/api";
import { useQuery, useQueryClient } from "@tanstack/react-query";
import { useEffect, useMemo } from "react";
import { QUERY_KEY_PERSISTENT_VOLUME_DETAIL } from "../../api/api.const";
import { GetPersistentVolumeByName, PersistentVolumeDetail } from "../../api/resources";
import { usePersistentVolumesUpdateEvents } from "../async-events/usePersistentVolumesUpdateEvents";

export function useGetPersistentVolumeByName(context: string, name: string) {
  const queryClient = useQueryClient();
  const latestPersistentVolumes = usePersistentVolumesUpdateEvents();

  const query = useQuery<PersistentVolumeDetail>({
    queryKey: [QUERY_KEY_PERSISTENT_VOLUME_DETAIL, { context, name }],
    queryFn: async () => GetPersistentVolumeByName(name),
    enabled: !!context && !!name,
    ...DEFAULT_QUERY_OPTIONS,
  });

  // The pushed event carries the lighter list DTO, not the full detail shape — invalidate
  // to refetch the detail rather than overwriting the cache with a mismatched shape.
  const pvKeyDependency = useMemo(() => {
    const matchedPv = latestPersistentVolumes.find((pv) => pv.Name === name);
    return matchedPv ? JSON.stringify(matchedPv) : null;
  }, [latestPersistentVolumes, name]);

  useEffect(() => {
    if (pvKeyDependency)
      queryClient.invalidateQueries({
        queryKey: [QUERY_KEY_PERSISTENT_VOLUME_DETAIL, { context, name }],
      });
  }, [pvKeyDependency, context, name, queryClient]);

  return query;
}

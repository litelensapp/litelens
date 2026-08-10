import { useQuery, useQueryClient } from "@tanstack/react-query";
import { QUERY_KEY_PERSISTENT_VOLUME_YAML } from "../../api/api.const";
import { DEFAULT_QUERY_OPTIONS } from "../../../../../../shared/api/api";
import { GetPersistentVolumeYAML } from "../../api/resources";
import { useEffect, useMemo } from "react";
import { usePersistentVolumesUpdateEvents } from "../async-events/usePersistentVolumesUpdateEvents";

export function useGetPersistentVolumeYAML(context: string, name: string, enabled = true) {
  const queryClient = useQueryClient();
  const latestPersistentVolumes = usePersistentVolumesUpdateEvents();

  const query = useQuery({
    queryKey: [QUERY_KEY_PERSISTENT_VOLUME_YAML, { context, name }],
    queryFn: () => GetPersistentVolumeYAML(name),
    ...DEFAULT_QUERY_OPTIONS,
    enabled: !!context && !!name && enabled,
  });

  // Invalidate YAML cache for this persistent volume when a matching persistent volume update is received.
  // Use a stable derived value (serialized persistent volume key) as dependency to avoid invalidating
  // on every unrelated event churn.
  const pvKeyDependency = useMemo(() => {
    const matchedPv = latestPersistentVolumes.find((pv) => pv.Name === name);
    // Serialize the persistent volume to a stable string: changes only when the persistent volume's content meaningfully changes.
    if (matchedPv) return JSON.stringify(matchedPv);
    return null;
  }, [latestPersistentVolumes, name]);

  useEffect(() => {
    if (pvKeyDependency)
      queryClient.invalidateQueries({
        queryKey: [QUERY_KEY_PERSISTENT_VOLUME_YAML, { context, name }],
      });
  }, [pvKeyDependency, context, name, queryClient]);

  return query;
}

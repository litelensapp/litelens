import { useQuery, useQueryClient } from "@tanstack/react-query";
import { QUERY_KEY_STORAGE_CLASS_YAML } from "../../api/api.const";
import { DEFAULT_QUERY_OPTIONS } from "../../../../../../shared/api/api";
import { GetStorageClassYAML } from "../../api/resources";
import { useEffect, useMemo } from "react";
import { useStorageClassesUpdateEvents } from "../async-events/useStorageClassesUpdateEvents";

export function useGetStorageClassYAML(context: string, name: string, enabled = true) {
  const queryClient = useQueryClient();
  const latestStorageClasses = useStorageClassesUpdateEvents();

  const query = useQuery({
    queryKey: [QUERY_KEY_STORAGE_CLASS_YAML, { context, name }],
    queryFn: () => GetStorageClassYAML(name),
    ...DEFAULT_QUERY_OPTIONS,
    enabled: !!context && !!name && enabled,
  });

  // Invalidate YAML cache for this storage class when a matching storage class update is received.
  // Use a stable derived value (serialized storage class key) as dependency to avoid invalidating
  // on every unrelated event churn.
  const scKeyDependency = useMemo(() => {
    const matchedSc = latestStorageClasses.find((sc) => sc.Name === name);
    // Serialize the storage class to a stable string: changes only when the storage class's content meaningfully changes.
    if (matchedSc) return JSON.stringify(matchedSc);
    return null;
  }, [latestStorageClasses, name]);

  useEffect(() => {
    if (scKeyDependency)
      queryClient.invalidateQueries({
        queryKey: [QUERY_KEY_STORAGE_CLASS_YAML, { context, name }],
      });
  }, [scKeyDependency, context, name, queryClient]);

  return query;
}

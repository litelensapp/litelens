import { DEFAULT_QUERY_OPTIONS } from "../../../../../../shared/api/api";
import { QUERY_KEY_PRIORITY_CLASS_YAML } from "../../api/api.const";
import { GetPriorityClassYAML } from "../../api/resources";
import { useQuery, useQueryClient } from "@tanstack/react-query";
import { useEffect, useMemo } from "react";
import { usePriorityClassesUpdateEvents } from "../async-events/usePriorityClassesUpdateEvents";

export function useGetPriorityClassYAML(context: string, name: string, enabled = true) {
  const queryClient = useQueryClient();
  const latestPriorityClasses = usePriorityClassesUpdateEvents();

  const query = useQuery({
    queryKey: [QUERY_KEY_PRIORITY_CLASS_YAML, { context, name }],
    queryFn: () => GetPriorityClassYAML(name),
    ...DEFAULT_QUERY_OPTIONS,
    enabled: !!context && !!name && enabled,
  });

  // Invalidate YAML cache for this priority class when a matching priority class update is received.
  // Use a stable derived value (serialized priority class key) as dependency to avoid invalidating
  // on every unrelated event churn.
  const pcKeyDependency = useMemo(() => {
    const matchedPc = latestPriorityClasses.find((pc) => pc.Name === name);
    // Serialize the priority class to a stable string: changes only when the priority class's content meaningfully changes.
    if (matchedPc) return JSON.stringify(matchedPc);
    return null;
  }, [latestPriorityClasses, name]);

  useEffect(() => {
    if (pcKeyDependency)
      queryClient.invalidateQueries({
        queryKey: [QUERY_KEY_PRIORITY_CLASS_YAML, { context, name }],
      });
  }, [pcKeyDependency, context, name, queryClient]);

  return query;
}

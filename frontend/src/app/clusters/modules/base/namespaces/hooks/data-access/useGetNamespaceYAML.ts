import { DEFAULT_QUERY_OPTIONS } from "../../../../../../shared/api/api";
import { QUERY_KEY_NAMESPACE_YAML } from "../../api/api.const";
import { GetNamespaceYAML } from "../../api/resources";
import { useQuery, useQueryClient } from "@tanstack/react-query";
import { useEffect, useMemo } from "react";
import { useNamespacesUpdateEvents } from "../async-events/useNamespacesUpdateEvents";

export function useGetNamespaceYAML(context: string, name: string, enabled = true) {
  const queryClient = useQueryClient();
  const latestNamespaces = useNamespacesUpdateEvents();

  const query = useQuery({
    queryKey: [QUERY_KEY_NAMESPACE_YAML, { context, name }],
    queryFn: () => GetNamespaceYAML(name),
    ...DEFAULT_QUERY_OPTIONS,
    enabled: !!context && !!name && enabled,
  });

  // Invalidate YAML cache for this namespace when a matching namespace update is received.
  // Use a stable derived value (serialized namespace key) as dependency to avoid invalidating
  // on every unrelated event churn.
  const namespaceKeyDependency = useMemo(() => {
    const matchedNamespace = latestNamespaces.find((n) => n.Name === name);
    // Serialize the namespace to a stable string: changes only when the namespace's content meaningfully changes.
    if (matchedNamespace) return JSON.stringify(matchedNamespace);
    return null;
  }, [latestNamespaces, name]);

  useEffect(() => {
    if (namespaceKeyDependency)
      queryClient.invalidateQueries({
        queryKey: [QUERY_KEY_NAMESPACE_YAML, { context, name }],
      });
  }, [namespaceKeyDependency, context, name, queryClient]);

  return query;
}

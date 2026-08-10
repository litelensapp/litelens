import { useQuery } from "@tanstack/react-query";
import { useMemo } from "react";
import { DEFAULT_QUERY_OPTIONS } from "../../../../../../shared/api/api";
import { QUERY_KEY_NAMESPACE_DETAIL } from "../../api/api.const";
import type { Namespace } from "../../api/resources";
import { GetNamespaceByName } from "../../api/resources";
import { useNamespacesUpdateEvents } from "../async-events/useNamespacesUpdateEvents";

export const useGetNamespaceDetail = (context: string, name: string) => {
  const latestNamespaces = useNamespacesUpdateEvents();

  const query = useQuery<Namespace, Error>({
    queryKey: [QUERY_KEY_NAMESPACE_DETAIL, { context, name }],
    queryFn: () => GetNamespaceByName(name),
    ...DEFAULT_QUERY_OPTIONS,
    enabled: !!context && !!name,
  });

  // Merge event-driven data: prefer matched namespace from latest event if available.
  const mergedData = useMemo(() => {
    const matchedNamespace = latestNamespaces.find((n) => n.Name === name);
    if (matchedNamespace) return matchedNamespace;
    return query.data;
  }, [latestNamespaces, query.data, name]);

  return {
    ...query,
    data: mergedData,
  };
};

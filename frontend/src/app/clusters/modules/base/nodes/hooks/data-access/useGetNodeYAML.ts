import { DEFAULT_QUERY_OPTIONS } from "../../../../../../shared/api/api";
import { QUERY_KEY_NODE_YAML } from "../../api/api.const";
import { GetNodeYAML } from "../../api/resources";
import { useQuery, useQueryClient } from "@tanstack/react-query";
import { useEffect, useMemo } from "react";
import { useNodesUpdateEvents } from "../async-events/useNodesUpdateEvents";

export function useGetNodeYAML(context: string, name: string, enabled = true) {
  const queryClient = useQueryClient();
  const latestNodes = useNodesUpdateEvents();

  const query = useQuery({
    queryKey: [QUERY_KEY_NODE_YAML, { context, name }],
    queryFn: () => GetNodeYAML(name),
    ...DEFAULT_QUERY_OPTIONS,
    enabled: !!context && !!name && enabled,
  });

  // Invalidate YAML cache for this node when a matching node update is received.
  // Use a stable derived value (serialized node key) as dependency to avoid invalidating
  // on every unrelated event churn.
  const nodeKeyDependency = useMemo(() => {
    const matchedNode = latestNodes.find((n) => n.Name === name);
    // Serialize the node to a stable string: changes only when the node's content meaningfully changes.
    if (matchedNode) return JSON.stringify(matchedNode);
    return null;
  }, [latestNodes, name]);

  useEffect(() => {
    if (nodeKeyDependency)
      queryClient.invalidateQueries({
        queryKey: [QUERY_KEY_NODE_YAML, { context, name }],
      });
  }, [nodeKeyDependency, context, name, queryClient]);

  return query;
}

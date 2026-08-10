import { DEFAULT_QUERY_OPTIONS } from "../../../../../../shared/api/api";
import { useQuery } from "@tanstack/react-query";
import { useMemo } from "react";
import { QUERY_KEY_NODE_DETAIL } from "../../api/api.const";
import type { Node } from "../../api/resources";
import { GetNodeByName } from "../../api/resources";
import { useNodesUpdateEvents } from "../async-events/useNodesUpdateEvents";

export const useGetNodeDetail = (context: string, name: string) => {
  const latestNodes = useNodesUpdateEvents();

  const query = useQuery<Node, Error>({
    queryKey: [QUERY_KEY_NODE_DETAIL, { context, name }],
    queryFn: () => GetNodeByName(name),
    ...DEFAULT_QUERY_OPTIONS,
    enabled: !!context && !!name,
  });

  // Merge event-driven data: prefer matched node from latest event if available.
  const mergedData = useMemo(() => {
    const matchedNode = latestNodes.find((n) => n.Name === name);
    if (matchedNode) return matchedNode;
    return query.data;
  }, [latestNodes, query.data, name]);

  return {
    ...query,
    data: mergedData,
  };
};

import { DEFAULT_QUERY_OPTIONS } from "../../../../../../shared/api/api";
import { QUERY_KEY_HPA_YAML } from "../../api/api.const";
import { GetHPAYAML } from "../../api/resources";
import { useQuery, useQueryClient } from "@tanstack/react-query";
import { useEffect, useMemo } from "react";
import { useHPAsUpdateEvents } from "../async-events/useHPAsUpdateEvents";

export function useGetHPAYAML(context: string, namespace: string, name: string, enabled = true) {
  const queryClient = useQueryClient();
  const latestHPAs = useHPAsUpdateEvents();

  const query = useQuery({
    queryKey: [QUERY_KEY_HPA_YAML, { context, namespace, name }],
    queryFn: () => GetHPAYAML(namespace, name),
    ...DEFAULT_QUERY_OPTIONS,
    enabled: !!context && !!namespace && !!name && enabled,
  });

  const hpaKeyDependency = useMemo(() => {
    const matchedHPA = latestHPAs.find((hpa) => hpa.Namespace === namespace && hpa.Name === name);
    if (matchedHPA) return JSON.stringify(matchedHPA);
    return null;
  }, [latestHPAs, namespace, name]);

  useEffect(() => {
    if (hpaKeyDependency)
      queryClient.invalidateQueries({
        queryKey: [QUERY_KEY_HPA_YAML, { context, namespace, name }],
      });
  }, [hpaKeyDependency, context, namespace, name, queryClient]);

  return query;
}

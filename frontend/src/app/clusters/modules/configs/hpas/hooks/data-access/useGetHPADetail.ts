import { DEFAULT_QUERY_OPTIONS } from "../../../../../../shared/api/api";
import { useQuery, useQueryClient } from "@tanstack/react-query";
import { useEffect, useMemo } from "react";
import { QUERY_KEY_HPA_DETAIL } from "../../api/api.const";
import type { HPADetail } from "../../api/resources";
import { GetHPAByName } from "../../api/resources";
import { useHPAsUpdateEvents } from "../async-events/useHPAsUpdateEvents";

export const useGetHPADetail = (context: string, namespace: string, name: string) => {
  const queryClient = useQueryClient();
  const latestHPAs = useHPAsUpdateEvents([namespace]);

  const query = useQuery<HPADetail, Error>({
    queryKey: [QUERY_KEY_HPA_DETAIL, { context, namespace, name }],
    queryFn: () => GetHPAByName(namespace, name),
    ...DEFAULT_QUERY_OPTIONS,
    enabled: !!context && !!namespace && !!name,
  });

  const hpaKeyDependency = useMemo(() => {
    const matchedHPA = latestHPAs.find((hpa) => hpa.Namespace === namespace && hpa.Name === name);
    return matchedHPA ? JSON.stringify(matchedHPA) : null;
  }, [latestHPAs, namespace, name]);

  useEffect(() => {
    if (hpaKeyDependency)
      queryClient.invalidateQueries({
        queryKey: [QUERY_KEY_HPA_DETAIL, { context, namespace, name }],
      });
  }, [hpaKeyDependency, context, namespace, name, queryClient]);

  return query;
};

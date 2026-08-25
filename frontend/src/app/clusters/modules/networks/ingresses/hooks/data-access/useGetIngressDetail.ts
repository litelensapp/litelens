import { DEFAULT_QUERY_OPTIONS } from "../../../../../../shared/api/api";
import { useQuery } from "@tanstack/react-query";
import { useMemo } from "react";
import { QUERY_KEY_INGRESS_DETAIL } from "../../api/api.const";
import type { Ingress, IngressDetail } from "../../api/resources";
import { GetIngressByName } from "../../api/resources";
import { useIngressesUpdateEvents } from "../async-events/useIngressesUpdateEvents";

export const useGetIngressDetail = (context: string, namespace: string, name: string) => {
  const latestIngresses = useIngressesUpdateEvents();

  const query = useQuery<IngressDetail, Error>({
    queryKey: [QUERY_KEY_INGRESS_DETAIL, { context, namespace, name }],
    queryFn: () => GetIngressByName(namespace, name),
    ...DEFAULT_QUERY_OPTIONS,
    enabled: !!context && !!namespace && !!name,
  });

  // Merge event-driven data: prefer matched ingress from latest event if available.
  // Note: event data is Ingress (summary), not IngressDetail; we use it if present.
  const mergedData = useMemo(() => {
    const matchedIngress = latestIngresses.find(
      (ing) => ing.Namespace === namespace && ing.Name === name
    ) as Ingress | IngressDetail | undefined;
    if (matchedIngress) return matchedIngress as IngressDetail;
    return query.data;
  }, [latestIngresses, query.data, namespace, name]);

  return {
    ...query,
    data: mergedData,
  };
};

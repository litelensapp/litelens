import { DEFAULT_QUERY_OPTIONS } from "../../../../../../shared/api/api";
import type { UseQueryCallback } from "@litelens/design-system";
import { useQuery } from "@tanstack/react-query";
import { useMemo } from "react";
import { QUERY_KEY_INGRESSES } from "../../api/api.const";
import type { Ingress } from "../../api/resources";
import { ListIngresses } from "../../api/resources";
import { useIngressesUpdateEvents } from "../async-events/useIngressesUpdateEvents";

export const useGetIngresses = (
  input: { context: string; namespace: string },
  callback?: UseQueryCallback<Ingress[]>
) => {
  const { context, namespace } = input;
  const latestIngresses = useIngressesUpdateEvents();

  const query = useQuery<Ingress[], Error>({
    queryKey: [QUERY_KEY_INGRESSES, { context, namespace }],
    queryFn: () => ListIngresses(namespace),
    ...DEFAULT_QUERY_OPTIONS,
    enabled: !!context,
  });

  // Merge event-driven data locally: prefer event-filtered ingresses over fetched data if available.
  // Filter cluster-wide event list to this hook's namespace (or include all if namespace === "").
  const mergedData = useMemo(() => {
    let baseData = query.data;
    if (latestIngresses.length)
      baseData =
        namespace === ""
          ? latestIngresses
          : latestIngresses.filter((ing) => ing.Namespace === namespace);
    return callback?.select ? callback.select(baseData) : baseData;
  }, [latestIngresses, query.data, namespace, callback]);

  return {
    ...query,
    data: mergedData,
  };
};

import { DEFAULT_QUERY_OPTIONS } from "../../../../../../shared/api/api";
import type { UseQueryCallback } from "@litelens/core";
import { useMemo } from "react";
import { useQuery } from "@tanstack/react-query";
import { QUERY_KEY_INGRESSES } from "../../api/api.const";
import type { Ingress } from "../../api/resources";
import { ListIngresses } from "../../api/resources";
import {
  getEffectiveNamespace,
  filterByNamespaces,
} from "../../../../../shared/utils/namespaceFiltering";
import { useIngressesUpdateEvents } from "../async-events/useIngressesUpdateEvents";

export const useGetIngresses = (
  input: { context: string; namespaces: string[] },
  callback?: UseQueryCallback<Ingress[]>
) => {
  const { context, namespaces } = input;
  const effectiveNamespace = getEffectiveNamespace(namespaces);
  const latestIngresses = useIngressesUpdateEvents();

  const query = useQuery<Ingress[], Error>({
    queryKey: [QUERY_KEY_INGRESSES, { context, namespaces }],
    queryFn: () => ListIngresses(effectiveNamespace),
    ...DEFAULT_QUERY_OPTIONS,
    enabled: !!context,
  });

  const mergedData = useMemo(() => {
    let baseData = query.data;
    if (latestIngresses.length) baseData = filterByNamespaces(latestIngresses, namespaces);
    return callback?.select ? callback.select(baseData) : baseData;
  }, [latestIngresses, query.data, namespaces, callback]);

  return { ...query, data: mergedData };
};

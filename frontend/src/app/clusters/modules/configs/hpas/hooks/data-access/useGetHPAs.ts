import { DEFAULT_QUERY_OPTIONS } from "../../../../../../shared/api/api";
import type { UseQueryCallback } from "@litelens/core";
import { useMemo } from "react";
import { useQuery } from "@tanstack/react-query";
import { QUERY_KEY_HPAS } from "../../api/api.const";
import type { HPA } from "../../api/resources";
import { ListHPAs } from "../../api/resources";
import {
  getEffectiveNamespace,
  filterByNamespaces,
} from "../../../../../shared/utils/namespaceFiltering";
import { useHPAsUpdateEvents } from "../async-events/useHPAsUpdateEvents";

export const useGetHPAs = (
  input: { context: string; namespaces: string[] },
  callback?: UseQueryCallback<HPA[]>
) => {
  const { context, namespaces } = input;
  const effectiveNamespace = getEffectiveNamespace(namespaces);
  const latestHPAs = useHPAsUpdateEvents(effectiveNamespace);

  const query = useQuery<HPA[], Error>({
    queryKey: [QUERY_KEY_HPAS, { context, namespaces }],
    queryFn: () => ListHPAs(effectiveNamespace),
    ...DEFAULT_QUERY_OPTIONS,
    enabled: !!context,
  });

  const mergedData = useMemo(() => {
    let baseData = query.data;
    if (latestHPAs.length) baseData = filterByNamespaces(latestHPAs, namespaces);
    return callback?.select ? callback.select(baseData) : baseData;
  }, [latestHPAs, query.data, namespaces, callback]);

  return { ...query, data: mergedData };
};

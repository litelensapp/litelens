import { DEFAULT_QUERY_OPTIONS } from "../../../../../../shared/api/api";
import type { UseQueryCallback } from "@litelens/design-system";
import { useMemo } from "react";
import { useQuery } from "@tanstack/react-query";
import { QUERY_KEY_LEASES } from "../../api/api.const";
import type { Lease } from "../../api/resources";
import { ListLeases } from "../../api/resources";
import {
  getEffectiveNamespace,
  filterByNamespaces,
} from "../../../../../shared/utils/namespaceFiltering";
import { useLeasesUpdateEvents } from "../async-events/useLeasesUpdateEvents";

export const useGetLeases = (
  input: { context: string; namespaces: string[] },
  callback?: UseQueryCallback<Lease[]>
) => {
  const { context, namespaces } = input;
  const effectiveNamespace = getEffectiveNamespace(namespaces);
  const latestLeases = useLeasesUpdateEvents(effectiveNamespace);

  const query = useQuery<Lease[], Error>({
    queryKey: [QUERY_KEY_LEASES, { context, namespaces }],
    queryFn: () => ListLeases(effectiveNamespace),
    ...DEFAULT_QUERY_OPTIONS,
    enabled: !!context,
  });

  const mergedData = useMemo(() => {
    let baseData = query.data;
    if (latestLeases.length) baseData = filterByNamespaces(latestLeases, namespaces);
    return callback?.select ? callback.select(baseData) : baseData;
  }, [latestLeases, query.data, namespaces, callback]);

  return { ...query, data: mergedData };
};

import { DEFAULT_QUERY_OPTIONS } from "../../../../../../shared/api/api";
import type { UseQueryCallback } from "@litelens/core";
import { useMemo } from "react";
import { useQuery } from "@tanstack/react-query";
import { QUERY_KEY_CONFIGMAPS } from "../../api/api.const";
import type { ConfigMap } from "../../api/resources";
import { ListConfigMaps } from "../../api/resources";
import { filterByNamespaces } from "../../../../../shared/utils/namespaceFiltering";
import { useConfigMapsUpdateEvents } from "../async-events/useConfigMapsUpdateEvents";

export const useGetConfigMaps = (
  input: { context: string; namespaces: string[] },
  callback?: UseQueryCallback<ConfigMap[]>
) => {
  const { context, namespaces } = input;
  const latestConfigMaps = useConfigMapsUpdateEvents(namespaces);

  const query = useQuery<ConfigMap[], Error>({
    queryKey: [QUERY_KEY_CONFIGMAPS, { context, namespaces }],
    queryFn: () => ListConfigMaps(namespaces),
    ...DEFAULT_QUERY_OPTIONS,
    enabled: !!context,
  });

  const mergedData = useMemo(() => {
    let baseData = query.data;
    if (latestConfigMaps.length) baseData = filterByNamespaces(latestConfigMaps, namespaces);
    return callback?.select ? callback.select(baseData) : baseData;
  }, [latestConfigMaps, query.data, namespaces, callback]);

  return { ...query, data: mergedData };
};

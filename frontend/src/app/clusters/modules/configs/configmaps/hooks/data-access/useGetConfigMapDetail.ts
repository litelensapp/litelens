import { useQuery } from "@tanstack/react-query";
import { useMemo } from "react";
import { DEFAULT_QUERY_OPTIONS } from "../../../../../../shared/api/api";
import { QUERY_KEY_CONFIGMAP_DETAIL } from "../../api/api.const";
import type { ConfigMap } from "../../api/resources";
import { GetConfigMapByName } from "../../api/resources";
import { useConfigMapsUpdateEvents } from "../async-events/useConfigMapsUpdateEvents";

export const useGetConfigMapDetail = (context: string, namespace: string, name: string) => {
  const latestConfigMaps = useConfigMapsUpdateEvents([namespace]);
  const query = useQuery<ConfigMap, Error>({
    queryKey: [QUERY_KEY_CONFIGMAP_DETAIL, { context, namespace, name }],
    queryFn: () => GetConfigMapByName(namespace, name),
    ...DEFAULT_QUERY_OPTIONS,
    enabled: !!context && !!namespace && !!name,
  });

  const mergedData = useMemo(() => {
    const matchedConfigMap = latestConfigMaps.find(
      (cm) => cm.Namespace === namespace && cm.Name === name
    );
    if (matchedConfigMap) return matchedConfigMap;
    return query.data;
  }, [latestConfigMaps, query.data, namespace, name]);

  return { ...query, data: mergedData };
};

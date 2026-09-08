import { useEffect, useMemo } from "react";
import { useQuery, useQueryClient } from "@tanstack/react-query";
import { QUERY_KEY_CONFIGMAP_YAML } from "../../api/api.const";
import { DEFAULT_QUERY_OPTIONS } from "../../../../../../shared/api/api";
import { GetConfigMapYAML } from "../../api/resources";
import { useConfigMapsUpdateEvents } from "../async-events/useConfigMapsUpdateEvents";

export function useGetConfigMapYAML(
  context: string,
  namespace: string,
  name: string,
  enabled = true
) {
  const queryClient = useQueryClient();
  const latestConfigMaps = useConfigMapsUpdateEvents();

  const query = useQuery({
    queryKey: [QUERY_KEY_CONFIGMAP_YAML, { context, namespace, name }],
    queryFn: () => GetConfigMapYAML(namespace, name),
    ...DEFAULT_QUERY_OPTIONS,
    enabled: !!context && !!namespace && !!name && enabled,
  });

  const configMapKeyDependency = useMemo(() => {
    const matchedConfigMap = latestConfigMaps.find(
      (cm) => cm.Namespace === namespace && cm.Name === name
    );
    if (matchedConfigMap) return JSON.stringify(matchedConfigMap);
    return null;
  }, [latestConfigMaps, namespace, name]);

  useEffect(() => {
    if (configMapKeyDependency) {
      queryClient.invalidateQueries({
        queryKey: [QUERY_KEY_CONFIGMAP_YAML, { context, namespace, name }],
      });
    }
  }, [configMapKeyDependency, context, namespace, name, queryClient]);

  return query;
}

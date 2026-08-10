import { DEFAULT_QUERY_OPTIONS } from "../../../../../../shared/api/api";
import type { UseQueryCallback } from "@litelens/design-system";
import { useMemo } from "react";
import { useQuery } from "@tanstack/react-query";
import { QUERY_KEY_CONFIGMAPS } from "../../api/api.const";
import type { ConfigMap } from "../../api/resources";
import { ListConfigMaps } from "../../api/resources";
import { useConfigMapsUpdateEvents } from "../async-events/useConfigMapsUpdateEvents";

export const useGetConfigMaps = (
  input: { context: string; namespace: string },
  callback?: UseQueryCallback<ConfigMap[]>
) => {
  const { context, namespace } = input;
  const latestConfigMaps = useConfigMapsUpdateEvents(namespace);
  const query = useQuery<ConfigMap[], Error>({
    queryKey: [QUERY_KEY_CONFIGMAPS, { context, namespace }],
    queryFn: () => ListConfigMaps(namespace),
    ...DEFAULT_QUERY_OPTIONS,
    enabled: !!context,
  });

  const mergedData = useMemo(() => {
    let baseData = query.data;
    if (latestConfigMaps.length)
      baseData =
        namespace === ""
          ? latestConfigMaps
          : latestConfigMaps.filter((cm) => cm.Namespace === namespace);
    return callback?.select ? callback.select(baseData) : baseData;
  }, [latestConfigMaps, query.data, namespace, callback]);

  return { ...query, data: mergedData };
};

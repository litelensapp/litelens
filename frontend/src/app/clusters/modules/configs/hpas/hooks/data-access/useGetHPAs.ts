import { DEFAULT_QUERY_OPTIONS } from "../../../../../../shared/api/api";
import type { UseQueryCallback } from "@litelens/design-system";
import { useQuery } from "@tanstack/react-query";
import { useMemo } from "react";
import { QUERY_KEY_HPAS } from "../../api/api.const";
import type { HPA } from "../../api/resources";
import { ListHPAs } from "../../api/resources";
import { useHPAsUpdateEvents } from "../async-events/useHPAsUpdateEvents";

export const useGetHPAs = (
  input: { context: string; namespace: string },
  callback?: UseQueryCallback<HPA[]>
) => {
  const { context, namespace } = input;
  const latestHPAs = useHPAsUpdateEvents(namespace);

  const query = useQuery<HPA[], Error>({
    queryKey: [QUERY_KEY_HPAS, { context, namespace }],
    queryFn: () => ListHPAs(namespace),
    ...DEFAULT_QUERY_OPTIONS,
    enabled: !!context,
  });

  const mergedData = useMemo(() => {
    let baseData = query.data;
    if (latestHPAs.length)
      baseData =
        namespace === "" ? latestHPAs : latestHPAs.filter((hpa) => hpa.Namespace === namespace);
    return callback?.select ? callback.select(baseData) : baseData;
  }, [latestHPAs, query.data, namespace, callback]);

  return { ...query, data: mergedData };
};

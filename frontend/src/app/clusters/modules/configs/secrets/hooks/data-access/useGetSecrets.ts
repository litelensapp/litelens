import { DEFAULT_QUERY_OPTIONS } from "../../../../../../shared/api/api";
import type { UseQueryCallback } from "@litelens/design-system";
import { useMemo } from "react";
import { useQuery } from "@tanstack/react-query";
import { QUERY_KEY_SECRETS } from "../../api/api.const";
import type { Secret } from "../../api/resources";
import { ListSecrets } from "../../api/resources";
import { useSecretsUpdateEvents } from "../async-events/useSecretsUpdateEvents";

export const useGetSecrets = (
  input: { context: string; namespace: string },
  callback?: UseQueryCallback<Secret[]>
) => {
  const { context, namespace } = input;
  const latestSecrets = useSecretsUpdateEvents(namespace);
  const query = useQuery<Secret[], Error>({
    queryKey: [QUERY_KEY_SECRETS, { context, namespace }],
    queryFn: () => ListSecrets(namespace),
    ...DEFAULT_QUERY_OPTIONS,
    enabled: !!context,
  });

  const mergedData = useMemo(() => {
    let baseData = query.data;
    if (latestSecrets.length)
      baseData =
        namespace === "" ? latestSecrets : latestSecrets.filter((s) => s.Namespace === namespace);
    return callback?.select ? callback.select(baseData) : baseData;
  }, [latestSecrets, query.data, namespace, callback]);

  return { ...query, data: mergedData };
};

import { DEFAULT_QUERY_OPTIONS } from "../../../../../../shared/api/api";
import type { UseQueryCallback } from "@litelens/core";
import { useMemo } from "react";
import { useQuery } from "@tanstack/react-query";
import { QUERY_KEY_SECRETS } from "../../api/api.const";
import type { Secret } from "../../api/resources";
import { ListSecrets } from "../../api/resources";
import { filterByNamespaces } from "../../../../../shared/utils/namespaceFiltering";
import { useSecretsUpdateEvents } from "../async-events/useSecretsUpdateEvents";

export const useGetSecrets = (
  input: { context: string; namespaces: string[] },
  callback?: UseQueryCallback<Secret[]>
) => {
  const { context, namespaces } = input;
  const latestSecrets = useSecretsUpdateEvents(namespaces);
  const query = useQuery<Secret[], Error>({
    queryKey: [QUERY_KEY_SECRETS, { context, namespaces }],
    queryFn: () => ListSecrets(namespaces),
    ...DEFAULT_QUERY_OPTIONS,
    enabled: !!context,
  });

  const mergedData = useMemo(() => {
    let baseData = query.data;
    if (latestSecrets.length) baseData = filterByNamespaces(latestSecrets, namespaces);
    return callback?.select ? callback.select(baseData) : baseData;
  }, [latestSecrets, query.data, namespaces, callback]);

  return { ...query, data: mergedData };
};

import { DEFAULT_QUERY_OPTIONS } from "../../../../../../shared/api/api";
import type { UseQueryCallback } from "@litelens/design-system";
import { useMemo } from "react";
import { useQuery } from "@tanstack/react-query";
import { QUERY_KEY_SECRETS } from "../../api/api.const";
import type { Secret } from "../../api/resources";
import { ListSecrets } from "../../api/resources";
import {
  getEffectiveNamespace,
  filterByNamespaces,
} from "../../../../../shared/utils/namespaceFiltering";
import { useSecretsUpdateEvents } from "../async-events/useSecretsUpdateEvents";

export const useGetSecrets = (
  input: { context: string; namespaces: string[] },
  callback?: UseQueryCallback<Secret[]>
) => {
  const { context, namespaces } = input;
  const effectiveNamespace = getEffectiveNamespace(namespaces);
  const latestSecrets = useSecretsUpdateEvents(effectiveNamespace);
  const query = useQuery<Secret[], Error>({
    queryKey: [QUERY_KEY_SECRETS, { context, namespaces }],
    queryFn: () => ListSecrets(effectiveNamespace),
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

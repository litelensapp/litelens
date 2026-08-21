import { DEFAULT_QUERY_OPTIONS } from "../../../../../../shared/api/api";
import { useQuery, useQueryClient } from "@tanstack/react-query";
import { useEffect, useMemo } from "react";
import { QUERY_KEY_SECRET_DETAIL } from "../../api/api.const";
import type { SecretDetail } from "../../api/resources";
import { GetSecretByName } from "../../api/resources";
import { useSecretsUpdateEvents } from "../async-events/useSecretsUpdateEvents";

export const useGetSecretDetail = (context: string, namespace: string, name: string) => {
  const queryClient = useQueryClient();
  const latestSecrets = useSecretsUpdateEvents([namespace]);
  const query = useQuery<SecretDetail, Error>({
    queryKey: [QUERY_KEY_SECRET_DETAIL, { context, namespace, name }],
    queryFn: () => GetSecretByName(namespace, name),
    ...DEFAULT_QUERY_OPTIONS,
    enabled: !!context && !!namespace && !!name,
  });

  const matchedSecret = useMemo(
    () => latestSecrets.find((s) => s.Namespace === namespace && s.Name === name),
    [latestSecrets, namespace, name]
  );

  useEffect(() => {
    if (matchedSecret) {
      queryClient.invalidateQueries({
        queryKey: [QUERY_KEY_SECRET_DETAIL, { context, namespace, name }],
      });
    }
  }, [matchedSecret, context, namespace, name, queryClient]);

  return query;
};

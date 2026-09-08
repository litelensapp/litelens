import { DEFAULT_QUERY_OPTIONS } from "../../../../../../shared/api/api";
import { QUERY_KEY_SECRET_YAML } from "../../api/api.const";
import { GetSecretYAML } from "../../api/resources";
import { useQuery, useQueryClient } from "@tanstack/react-query";
import { useEffect, useMemo } from "react";
import { useSecretsUpdateEvents } from "../async-events/useSecretsUpdateEvents";

export function useGetSecretYAML(context: string, namespace: string, name: string, enabled = true) {
  const queryClient = useQueryClient();
  const latestSecrets = useSecretsUpdateEvents();

  const query = useQuery({
    queryKey: [QUERY_KEY_SECRET_YAML, { context, namespace, name }],
    queryFn: () => GetSecretYAML(namespace, name),
    ...DEFAULT_QUERY_OPTIONS,
    enabled: !!context && !!namespace && !!name && enabled,
  });

  const secretKeyDependency = useMemo(() => {
    const matchedSecret = latestSecrets.find((s) => s.Namespace === namespace && s.Name === name);
    if (matchedSecret) return JSON.stringify(matchedSecret);
    return null;
  }, [latestSecrets, namespace, name]);

  useEffect(() => {
    if (secretKeyDependency)
      queryClient.invalidateQueries({
        queryKey: [QUERY_KEY_SECRET_YAML, { context, namespace, name }],
      });
  }, [secretKeyDependency, context, namespace, name, queryClient]);

  return query;
}

import { useQuery, useQueryClient } from "@tanstack/react-query";
import { useEffect, useMemo } from "react";
import { QUERY_KEY_VALIDATING_WEBHOOK_CONFIG_YAML } from "../../api/api.const";
import { DEFAULT_QUERY_OPTIONS } from "../../../../../../shared/api/api";
import { GetValidatingWebhookConfigYAML } from "../../api/resources";
import { useValidatingWebhookConfigsUpdateEvents } from "../async-events/useValidatingWebhookConfigsUpdateEvents";

export function useGetValidatingWebhookConfigYAML(context: string, name: string, enabled = true) {
  const latestValidatingWebhookConfigs = useValidatingWebhookConfigsUpdateEvents();
  const queryClient = useQueryClient();

  const query = useQuery({
    queryKey: [QUERY_KEY_VALIDATING_WEBHOOK_CONFIG_YAML, { context, name }],
    queryFn: () => GetValidatingWebhookConfigYAML(name),
    ...DEFAULT_QUERY_OPTIONS,
    enabled: !!context && !!name && enabled,
  });

  const validatingWebhookConfigKeyDependency = useMemo(() => {
    const matchedValidatingWebhookConfig = latestValidatingWebhookConfigs.find(
      (vwc) => vwc.Name === name
    );
    if (matchedValidatingWebhookConfig) return JSON.stringify(matchedValidatingWebhookConfig);
    return null;
  }, [latestValidatingWebhookConfigs, name]);

  useEffect(() => {
    if (validatingWebhookConfigKeyDependency)
      queryClient.invalidateQueries({
        queryKey: [QUERY_KEY_VALIDATING_WEBHOOK_CONFIG_YAML, { context, name }],
      });
  }, [validatingWebhookConfigKeyDependency, context, name, queryClient]);

  return query;
}

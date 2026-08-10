import { useQuery, useQueryClient } from "@tanstack/react-query";
import { useEffect } from "react";
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

  useEffect(() => {
    const matchedValidatingWebhookConfig = latestValidatingWebhookConfigs.some(
      (vwc) => vwc.Name === name
    );
    if (matchedValidatingWebhookConfig)
      queryClient.invalidateQueries({
        queryKey: [QUERY_KEY_VALIDATING_WEBHOOK_CONFIG_YAML, { context, name }],
      });
  }, [latestValidatingWebhookConfigs, context, name, queryClient]);

  return query;
}

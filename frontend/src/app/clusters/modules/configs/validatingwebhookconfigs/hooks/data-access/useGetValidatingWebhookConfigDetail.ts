import { DEFAULT_QUERY_OPTIONS } from "../../../../../../shared/api/api";
import { useQuery } from "@tanstack/react-query";
import { useMemo } from "react";
import { QUERY_KEY_VALIDATING_WEBHOOK_CONFIG_DETAIL } from "../../api/api.const";
import type { ValidatingWebhookConfigDetail } from "../../api/resources";
import { GetValidatingWebhookConfigByName } from "../../api/resources";
import { useValidatingWebhookConfigsUpdateEvents } from "../async-events/useValidatingWebhookConfigsUpdateEvents";

export const useGetValidatingWebhookConfigDetail = (context: string, name: string) => {
  const latestValidatingWebhookConfigs = useValidatingWebhookConfigsUpdateEvents();
  const query = useQuery<ValidatingWebhookConfigDetail, Error>({
    queryKey: [QUERY_KEY_VALIDATING_WEBHOOK_CONFIG_DETAIL, { context, name }],
    queryFn: () => GetValidatingWebhookConfigByName(name),
    ...DEFAULT_QUERY_OPTIONS,
    enabled: !!context && !!name,
  });

  const mergedData = useMemo(() => {
    const matchedValidatingWebhookConfig = latestValidatingWebhookConfigs.find(
      (vwc) => vwc.Name === name
    ) as ValidatingWebhookConfigDetail | undefined;
    if (matchedValidatingWebhookConfig) return matchedValidatingWebhookConfig;
    return query.data;
  }, [latestValidatingWebhookConfigs, query.data, name]);

  return { ...query, data: mergedData };
};

import { useEffect, useState } from "react";
import { EventsOn } from "@wailsjs/runtime/runtime";
import type { ValidatingWebhookConfig } from "../../api/resources";

export function useValidatingWebhookConfigsUpdateEvents(): ValidatingWebhookConfig[] {
  const [latestValidatingWebhookConfigs, setLatestValidatingWebhookConfigs] = useState<
    ValidatingWebhookConfig[]
  >([]);
  useEffect(() => {
    return EventsOn("validatingwebhookconfigs:update", (data: ValidatingWebhookConfig[]) =>
      setLatestValidatingWebhookConfigs(data)
    );
  }, []);
  return latestValidatingWebhookConfigs;
}

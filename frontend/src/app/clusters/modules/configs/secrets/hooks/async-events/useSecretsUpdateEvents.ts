import { startTransition, useEffect, useState } from "react";
import { EventsOn } from "@wailsjs/runtime/runtime";
import type { Secret } from "../../api/resources";

// Data-only event hook: tracks the latest pushed secrets in local state.
// Called directly from secret data-access hooks (useGetSecrets, useGetSecretDetail, useGetSecretYAML)
// to merge event-driven data locally without cache-wide side effects.
// Pass a namespace to subscribe to the backend's namespace-scoped channel
// ("secrets:{namespace}:update") instead of the cluster-wide "secrets:update" broadcast.
export function useSecretsUpdateEvents(namespace = ""): Secret[] {
  const [latestSecrets, setLatestSecrets] = useState<Secret[]>([]);
  const [prevNamespace, setPrevNamespace] = useState(namespace);

  // Reset stale data from the previous namespace's channel before re-subscribing.
  if (namespace !== prevNamespace) {
    setPrevNamespace(namespace);
    setLatestSecrets([]);
  }

  useEffect(() => {
    const eventName = namespace ? `secrets:${namespace}:update` : "secrets:update";
    return EventsOn(eventName, (data: Secret[]) => {
      startTransition(() => {
        setLatestSecrets(data);
      });
    });
  }, [namespace]);
  return latestSecrets;
}

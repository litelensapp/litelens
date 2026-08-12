import { startTransition, useEffect, useState } from "react";
import { EventsOn } from "@wailsjs/runtime/runtime";
import type { NetworkPolicy } from "../../api/resources";

export function useNetworkPoliciesUpdateEvents(): NetworkPolicy[] {
  const [latestNetworkPolicies, setLatestNetworkPolicies] = useState<NetworkPolicy[]>([]);
  useEffect(() => {
    return EventsOn("networkpolicies:update", (data: NetworkPolicy[]) => {
      startTransition(() => {
        setLatestNetworkPolicies(data);
      });
    });
  }, []);
  return latestNetworkPolicies;
}

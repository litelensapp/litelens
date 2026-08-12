import { startTransition, useEffect, useState } from "react";
import { EventsOn } from "@wailsjs/runtime/runtime";
import type { Endpoint } from "../../api/resources";

// Data-only event hook: tracks the latest pushed endpoints in local state.
// Called directly from endpoint data-access hooks (useGetEndpoints, useGetEndpointDetail, useGetEndpointYAML)
// to merge event-driven data locally without cache-wide side effects.
export function useEndpointsUpdateEvents(): Endpoint[] {
  const [latestEndpoints, setLatestEndpoints] = useState<Endpoint[]>([]);
  useEffect(() => {
    return EventsOn("endpoints:update", (data: Endpoint[]) => {
      startTransition(() => {
        setLatestEndpoints(data);
      });
    });
  }, []);
  return latestEndpoints;
}

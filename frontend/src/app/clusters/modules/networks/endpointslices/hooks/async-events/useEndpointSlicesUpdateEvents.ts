import { startTransition, useEffect, useState } from "react";
import { EventsOn } from "@wailsjs/runtime/runtime";
import type { EndpointSlice } from "../../api/resources";

// Data-only event hook: tracks the latest pushed endpoint slices in local state.
// Called directly from endpoint slice data-access hooks (useGetEndpointSlices, useGetEndpointSliceByName, useGetEndpointSliceYAML)
// to merge event-driven data locally without cache-wide side effects.
export function useEndpointSlicesUpdateEvents(): EndpointSlice[] {
  const [latestEndpointSlices, setLatestEndpointSlices] = useState<EndpointSlice[]>([]);
  useEffect(() => {
    return EventsOn("endpointslices:update", (data: EndpointSlice[]) => {
      startTransition(() => {
        setLatestEndpointSlices(data);
      });
    });
  }, []);
  return latestEndpointSlices;
}

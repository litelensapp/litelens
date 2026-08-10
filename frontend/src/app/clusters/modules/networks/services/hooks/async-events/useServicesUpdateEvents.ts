import { useEffect, useState } from "react";
import { EventsOn } from "@wailsjs/runtime/runtime";
import type { Service } from "../../api/resources";

// Data-only event hook: tracks the latest pushed services in local state.
// Called directly from service data-access hooks (useGetServices, useGetServiceDetail, useGetServiceYAML)
// to merge event-driven data locally without cache-wide side effects.
export function useServicesUpdateEvents(): Service[] {
  const [latestServices, setLatestServices] = useState<Service[]>([]);
  useEffect(() => {
    return EventsOn("services:update", (data: Service[]) => setLatestServices(data));
  }, []);
  return latestServices;
}

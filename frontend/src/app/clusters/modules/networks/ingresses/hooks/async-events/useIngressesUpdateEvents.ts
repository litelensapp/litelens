import { startTransition, useEffect, useState } from "react";
import { EventsOn } from "@wailsjs/runtime/runtime";
import type { Ingress } from "../../api/resources";

// Data-only event hook: tracks the latest pushed ingresses in local state.
// Called directly from ingress data-access hooks (useGetIngresses, useGetIngressDetail, useGetIngressYAML)
// to merge event-driven data locally without cache-wide side effects.
export function useIngressesUpdateEvents(): Ingress[] {
  const [latestIngresses, setLatestIngresses] = useState<Ingress[]>([]);
  useEffect(() => {
    return EventsOn("ingresses:update", (data: Ingress[]) => {
      startTransition(() => {
        setLatestIngresses(data);
      });
    });
  }, []);
  return latestIngresses;
}

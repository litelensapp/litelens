import { useEffect, useState, startTransition } from "react";
import { EventsOn } from "@wailsjs/runtime/runtime";
import type { Deployment } from "../../api/resources";

export function useDeploymentsUpdateEvents(): Deployment[] {
  const [latestDeployments, setLatestDeployments] = useState<Deployment[]>([]);
  useEffect(() => {
    return EventsOn("deployments:update", (data: Deployment[]) => {
      startTransition(() => {
        setLatestDeployments(data);
      });
    });
  }, []);
  return latestDeployments;
}

import { useEffect, useState } from "react";
import { EventsOn } from "@wailsjs/runtime/runtime";
import type { Deployment } from "../../api/resources";

export function useDeploymentsUpdateEvents(): Deployment[] {
  const [latestDeployments, setLatestDeployments] = useState<Deployment[]>([]);
  useEffect(() => {
    return EventsOn("deployments:update", (data: Deployment[]) => setLatestDeployments(data));
  }, []);
  return latestDeployments;
}

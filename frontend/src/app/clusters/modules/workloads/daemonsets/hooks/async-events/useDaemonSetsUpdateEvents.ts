import { EventsOn } from "@wailsjs/runtime/runtime";
import { useEffect, useState } from "react";
import type { DaemonSet } from "../../api/resources";

export function useDaemonSetsUpdateEvents(): DaemonSet[] {
  const [latestDaemonSets, setLatestDaemonSets] = useState<DaemonSet[]>([]);
  useEffect(() => {
    return EventsOn("daemonsets:update", (data: DaemonSet[]) => setLatestDaemonSets(data));
  }, []);
  return latestDaemonSets;
}

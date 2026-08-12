import { EventsOn } from "@wailsjs/runtime/runtime";
import { useEffect, useState, startTransition } from "react";
import type { DaemonSet } from "../../api/resources";

export function useDaemonSetsUpdateEvents(): DaemonSet[] {
  const [latestDaemonSets, setLatestDaemonSets] = useState<DaemonSet[]>([]);
  useEffect(() => {
    return EventsOn("daemonsets:update", (data: DaemonSet[]) => {
      startTransition(() => {
        setLatestDaemonSets(data);
      });
    });
  }, []);
  return latestDaemonSets;
}

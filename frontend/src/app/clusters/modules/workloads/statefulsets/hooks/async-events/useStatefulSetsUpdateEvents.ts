import { useEffect, useState, startTransition } from "react";
import { EventsOn } from "@wailsjs/runtime/runtime";
import type { StatefulSet } from "../../api/resources";

export function useStatefulSetsUpdateEvents(): StatefulSet[] {
  const [latestStatefulSets, setLatestStatefulSets] = useState<StatefulSet[]>([]);
  useEffect(() => {
    return EventsOn("statefulsets:update", (data: StatefulSet[]) => {
      startTransition(() => {
        setLatestStatefulSets(data);
      });
    });
  }, []);
  return latestStatefulSets;
}

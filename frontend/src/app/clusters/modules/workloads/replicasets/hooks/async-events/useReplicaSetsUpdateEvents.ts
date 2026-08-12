import { useEffect, useState, startTransition } from "react";
import { EventsOn } from "@wailsjs/runtime/runtime";
import type { ReplicaSet } from "../../api/resources";

export function useReplicaSetsUpdateEvents(): ReplicaSet[] {
  const [latestReplicaSets, setLatestReplicaSets] = useState<ReplicaSet[]>([]);
  useEffect(() => {
    return EventsOn("replicasets:update", (data: ReplicaSet[]) => {
      startTransition(() => {
        setLatestReplicaSets(data);
      });
    });
  }, []);
  return latestReplicaSets;
}

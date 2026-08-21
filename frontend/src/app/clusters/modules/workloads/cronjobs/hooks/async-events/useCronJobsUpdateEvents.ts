import { useEffect, useState, startTransition } from "react";
import { EventsOn } from "@wailsjs/runtime/runtime";
import type { CronJob } from "../../api/resources";
import { mergeNamespaceScopedData } from "../../../../../shared/utils/eventMerging";

export function useCronJobsUpdateEvents(namespaces: string[] = []): CronJob[] {
  const [latestCronJobs, setlatestCronJobs] = useState<CronJob[]>([]);
  const [prevNamespaces, setPrevNamespaces] = useState(namespaces);

  // When namespace selection changes, filter down accumulated state to only selected namespaces.
  if (JSON.stringify(prevNamespaces) !== JSON.stringify(namespaces)) {
    setPrevNamespaces(namespaces);
    if (namespaces.length > 0) {
      const namespacesSet = new Set(namespaces);
      setlatestCronJobs((prev) => prev.filter((item) => namespacesSet.has(item.Namespace)));
    } else {
      setlatestCronJobs([]);
    }
  }

  useEffect(() => {
    if (namespaces.length === 0) {
      return EventsOn("cronjobs:update", (data: CronJob[]) => {
        startTransition(() => {
          setlatestCronJobs(data);
        });
      });
    }

    const unsubscribers: Array<() => void> = [];
    for (const ns of namespaces) {
      const eventName = `cronjobs:${ns}:update`;
      const unsubscriber = EventsOn(eventName, (data: CronJob[]) => {
        startTransition(() => {
          setlatestCronJobs((prev) => mergeNamespaceScopedData(prev, data, ns));
        });
      });
      unsubscribers.push(unsubscriber);
    }

    return () => {
      unsubscribers.forEach((unsub) => unsub());
    };
  }, [namespaces]);
  return latestCronJobs;
}

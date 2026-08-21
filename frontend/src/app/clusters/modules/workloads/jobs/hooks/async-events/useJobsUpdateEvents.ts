import { startTransition, useEffect, useState } from "react";
import { EventsOn } from "@wailsjs/runtime/runtime";
import type { Job } from "../../api/resources";
import { mergeNamespaceScopedData } from "../../../../../shared/utils/eventMerging";

// Data-only event hook: tracks the latest pushed jobs in local state.
// Called directly from job data-access hooks (useGetJobs, useGetJobDetail, useGetJobYAML)
// to merge event-driven data locally without cache-wide side effects.
// Pass namespaces to subscribe to the backend's namespace-scoped channels
// ("jobs:{namespace}:update" for each namespace) instead of the cluster-wide "jobs:update" broadcast.
export function useJobsUpdateEvents(namespaces: string[] = []): Job[] {
  const [latestJobs, setLatestJobs] = useState<Job[]>([]);
  const [prevNamespaces, setPrevNamespaces] = useState(namespaces);

  // When namespace selection changes, filter down accumulated state to only selected namespaces.
  if (JSON.stringify(prevNamespaces) !== JSON.stringify(namespaces)) {
    setPrevNamespaces(namespaces);
    if (namespaces.length > 0) {
      const namespacesSet = new Set(namespaces);
      setLatestJobs((prev) => prev.filter((job) => namespacesSet.has(job.Namespace)));
    } else {
      setLatestJobs([]);
    }
  }

  useEffect(() => {
    if (namespaces.length === 0) {
      return EventsOn("jobs:update", (data: Job[]) => {
        startTransition(() => {
          setLatestJobs(data);
        });
      });
    }

    const unsubscribers: Array<() => void> = [];
    for (const ns of namespaces) {
      const eventName = `jobs:${ns}:update`;
      const unsubscriber = EventsOn(eventName, (data: Job[]) => {
        startTransition(() => {
          setLatestJobs((prev) => mergeNamespaceScopedData(prev, data, ns));
        });
      });
      unsubscribers.push(unsubscriber);
    }

    return () => {
      unsubscribers.forEach((unsub) => unsub());
    };
  }, [namespaces]);
  return latestJobs;
}

import { startTransition, useEffect, useState } from "react";
import { EventsOn } from "@wailsjs/runtime/runtime";
import type { Job } from "../../api/resources";

// Data-only event hook: tracks the latest pushed jobs in local state.
// Called directly from job data-access hooks (useGetJobs, useGetJobDetail, useGetJobYAML)
// to merge event-driven data locally without cache-wide side effects.
// Pass a namespace to subscribe to the backend's namespace-scoped channel
// ("jobs:{namespace}:update") instead of the cluster-wide "jobs:update" broadcast.
export function useJobsUpdateEvents(namespace = ""): Job[] {
  const [latestJobs, setLatestJobs] = useState<Job[]>([]);
  const [prevNamespace, setPrevNamespace] = useState(namespace);

  // Reset stale data from the previous namespace's channel before re-subscribing.
  if (namespace !== prevNamespace) {
    setPrevNamespace(namespace);
    setLatestJobs([]);
  }

  useEffect(() => {
    const eventName = namespace ? `jobs:${namespace}:update` : "jobs:update";
    return EventsOn(eventName, (data: Job[]) => {
      startTransition(() => {
        setLatestJobs(data);
      });
    });
  }, [namespace]);
  return latestJobs;
}

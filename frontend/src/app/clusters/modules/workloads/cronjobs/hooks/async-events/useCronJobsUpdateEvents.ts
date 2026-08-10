import { EventsOn } from "@wailsjs/runtime/runtime";
import { useEffect, useState } from "react";
import type { CronJob } from "../../api/resources";

export function useCronJobsUpdateEvents(): CronJob[] {
  const [latestCronJobs, setLatestCronJobs] = useState<CronJob[]>([]);
  useEffect(() => {
    return EventsOn("cronjobs:update", (data: CronJob[]) => setLatestCronJobs(data));
  }, []);
  return latestCronJobs;
}

import { Divider, DonutChart, ResourceLink } from "@litelens/design-system";
import { FC } from "react";
import { useGetCronJobs } from "../workloads/cronjobs/hooks/data-access/useGetCronJobs";
import { useGetDaemonSets } from "../workloads/daemonsets/hooks/data-access/useGetDaemonSets";
import { useGetDeployments } from "../workloads/deployments/hooks/data-access/useGetDeployments";
import { useGetJobs } from "../workloads/jobs/hooks/data-access/useGetJobs";
import { useGetPods } from "../workloads/pods/hooks/data-access/useGetPods";
import { useGetReplicaSets } from "../workloads/replicasets/hooks/data-access/useGetReplicaSets";
import { useGetStatefulSets } from "../workloads/statefulsets/hooks/data-access/useGetStatefulSets";
import { useGetWarningEvents } from "../base/events/hooks/data-access/useGetWarningEvents";
import { useMainLayoutContext } from "../../MainLayoutContext";
import { EventsTable } from "../base/events/components/EventsTable";
import { ViewType } from "../../navConfig";

function parsePodsStr(str: string): { ready: number; desired: number } {
  const [r, d] = str.split("/").map(Number);
  return { ready: r ?? 0, desired: d ?? 0 };
}

interface OverviewViewProps {
  onNavigateToView?: (view: ViewType) => void;
}

export const OverviewView: FC<OverviewViewProps> = ({ onNavigateToView }) => {
  const { activeContext, namespace } = useMainLayoutContext();

  const { data: pods = [] } = useGetPods({ context: activeContext, namespace });
  const { data: deployments = [] } = useGetDeployments({ context: activeContext, namespace });
  const { data: daemonSets = [] } = useGetDaemonSets({ context: activeContext, namespace });
  const { data: statefulSets = [] } = useGetStatefulSets({ context: activeContext, namespace });
  const { data: replicaSets = [] } = useGetReplicaSets({ context: activeContext, namespace });
  const { data: jobs = [] } = useGetJobs({ context: activeContext, namespace });
  const { data: cronJobs = [] } = useGetCronJobs({ context: activeContext, namespace });
  const { data: warningEvents = [] } = useGetWarningEvents({ context: activeContext, namespace });

  // Pods
  const podCounts = pods.reduce<Record<string, number>>((acc, p) => {
    acc[p.Status.toLowerCase()] = (acc[p.Status.toLowerCase()] ?? 0) + 1;
    return acc;
  }, {});
  const runningPods = podCounts["running"] ?? 0;
  const failedPods = podCounts["failed"] ?? 0;
  const succeededPods = podCounts["succeeded"] ?? 0;
  const pendingPods = podCounts["pending"] ?? 0;
  const evictedPods = podCounts["evicted"] ?? 0;

  // Deployments
  const runningDeployments = deployments.filter((d) => {
    const { ready, desired } = parsePodsStr(d.Pods);
    return desired > 0 && ready >= desired;
  }).length;
  const pendingDeployments = deployments.length - runningDeployments;

  // DaemonSets
  const runningDaemonSets = daemonSets.filter((d) => {
    const { ready, desired } = parsePodsStr(d.Pods);
    return desired > 0 && ready >= desired;
  }).length;
  const pendingDaemonSets = daemonSets.length - runningDaemonSets;

  // StatefulSets
  const runningStatefulSets = statefulSets.filter((s) => {
    const { ready, desired } = parsePodsStr(s.Pods);
    return desired > 0 && ready >= desired;
  }).length;
  const pendingStatefulSets = statefulSets.length - runningStatefulSets;

  // ReplicaSets — "running" when Ready >= Desired (includes 0/0 inactive RS)
  const runningReplicaSets = replicaSets.filter((r) => r.Ready >= r.Desired).length;
  const pendingReplicaSets = replicaSets.length - runningReplicaSets;

  // Jobs
  const succeededJobs = jobs.filter((j) =>
    j.Conditions?.map((c) => c.toLowerCase()).includes("complete")
  ).length;
  const failedJobs = jobs.filter((j) =>
    j.Conditions?.map((c) => c.toLowerCase()).includes("failed")
  ).length;
  const pendingJobs = jobs.length - succeededJobs - failedJobs;

  // CronJobs
  const scheduledCronJobs = cronJobs.filter((c) => !c.Suspend).length;
  const suspendedCronJobs = cronJobs.filter((c) => c.Suspend).length;

  return (
    <div className="flex h-full flex-col gap-6">
      <div className="flex flex-wrap gap-8">
        <DonutChart
          label="Pods"
          total={pods.length}
          running={runningPods}
          pending={pendingPods}
          failed={failedPods}
          items={[
            { label: "Succeeded", color: "green", count: succeededPods },
            { label: "Running", color: "green", count: runningPods },
            { label: "Pending", color: "amber", count: pendingPods },
            { label: "Failed", color: "red", count: failedPods },
            { label: "Evicted", color: "red", count: evictedPods },
          ]}
          onNavigate={() => onNavigateToView?.("pods")}
        />
        <DonutChart
          label="Deployments"
          total={deployments.length}
          running={runningDeployments}
          pending={pendingDeployments}
          items={[
            { label: "Running", color: "green", count: runningDeployments },
            { label: "Pending", color: "amber", count: pendingDeployments },
          ]}
          onNavigate={() => onNavigateToView?.("deployments")}
        />
        <DonutChart
          label="Daemon Sets"
          total={daemonSets.length}
          running={runningDaemonSets}
          pending={pendingDaemonSets}
          items={[
            { label: "Running", color: "green", count: runningDaemonSets },
            { label: "Pending", color: "amber", count: pendingDaemonSets },
          ]}
          onNavigate={() => onNavigateToView?.("daemonsets")}
        />
        <DonutChart
          label="Stateful Sets"
          total={statefulSets.length}
          running={runningStatefulSets}
          pending={pendingStatefulSets}
          items={[
            { label: "Running", color: "green", count: runningStatefulSets },
            { label: "Pending", color: "amber", count: pendingStatefulSets },
          ]}
          onNavigate={() => onNavigateToView?.("statefulsets")}
        />
        <DonutChart
          label="Replica Sets"
          total={replicaSets.length}
          running={runningReplicaSets}
          pending={pendingReplicaSets}
          items={[
            { label: "Running", color: "green", count: runningReplicaSets },
            { label: "Pending", color: "amber", count: pendingReplicaSets },
          ]}
          onNavigate={() => onNavigateToView?.("replicasets")}
        />
        <DonutChart
          label="Jobs"
          total={jobs.length}
          running={succeededJobs}
          pending={pendingJobs}
          failed={failedJobs}
          items={[
            { label: "Succeeded", color: "green", count: succeededJobs },
            { label: "Failed", color: "red", count: failedJobs },
            { label: "Pending", color: "amber", count: pendingJobs },
          ]}
          onNavigate={() => onNavigateToView?.("jobs")}
        />
        <DonutChart
          label="Cron Jobs"
          total={cronJobs.length}
          running={scheduledCronJobs}
          items={[
            { label: "Scheduled", color: "green", count: scheduledCronJobs },
            { label: "Suspended", color: "amber", count: suspendedCronJobs },
          ]}
          onNavigate={() => onNavigateToView?.("cronjobs")}
        />
      </div>

      <Divider />

      <div className="flex min-h-0 flex-1 flex-col gap-3">
        <span className="text-left text-sm font-medium">
          <ResourceLink onClick={() => onNavigateToView?.("events")}>
            Warning Events ({warningEvents.length})
          </ResourceLink>
        </span>
        <EventsTable events={warningEvents} />
      </div>
    </div>
  );
};

import { CpuIcon, LayersIcon, NetworkIcon, ScrollTextIcon } from "@litelens/design-system";
import { FC } from "react";
import logo from "../../../assets/images/logo-universal.png";

const FEATURES = [
  { icon: LayersIcon, label: "Workloads", desc: "Pods, Deployments, DaemonSets, StatefulSets" },
  { icon: NetworkIcon, label: "Networking", desc: "Services, Ingresses, NetworkIcon Policies" },
  { icon: CpuIcon, label: "Nodes", desc: "CPU, memory and disk usage at a glance" },
  { icon: ScrollTextIcon, label: "Events", desc: "Cluster-wide event stream in real time" },
];

export const WelcomeView: FC = () => (
  <div className="flex flex-1 flex-col items-center justify-center gap-10 p-10">
    <div className="flex flex-col items-center gap-2 text-center">
      <div className="bg-primary/10 text-primary mb-2 flex h-16 w-16 items-center justify-center rounded-2xl">
        <img src={logo} alt="LiteLens" className="mb-4 h-16 w-16 rounded-2xl" />
      </div>
      <h1 className="text-foreground text-2xl font-bold tracking-tight">LiteLens</h1>
      <p className="text-muted-foreground text-sm">Lightweight Kubernetes Cluster Manager</p>
    </div>

    <div className="text-center">
      <p className="text-muted-foreground text-sm">
        Select a cluster from the sidebar to connect and start exploring.
      </p>
    </div>

    <div className="grid w-full max-w-lg grid-cols-2 gap-3">
      {FEATURES.map(({ icon: Icon, label, desc }) => (
        <div key={label} className="bg-muted/30 border-border rounded-lg border p-4">
          <div className="mb-2 flex items-center gap-2">
            <Icon className="text-primary h-4 w-4 shrink-0" />
            <span className="text-foreground text-sm font-medium">{label}</span>
          </div>
          <p className="text-muted-foreground text-left text-xs">{desc}</p>
        </div>
      ))}
    </div>
  </div>
);

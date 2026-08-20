import type { ManagedField } from "./shared";

export interface ReplicaSet {
  Name: string;
  Namespace: string;
  Desired: number;
  Current: number;
  Ready: number;
  Age: string;
  OwnerName: string;
  CreatedAt: string;
  OwnerKind: string;
  Labels: Record<string, string>;
  Annotations: Record<string, string>;
  ManagedFields: ManagedField[];
  Selector: string;
  NodeSelector: string;
  Images: string[];
  ReplicasDetail: string;
  Tolerations: number;
  Affinities: number;
  PodStatus: string;
}

export interface ReplicaSetSummary {
  Running: number;
  Pending: number;
}

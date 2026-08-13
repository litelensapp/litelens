import type { ManagedField } from "../../../../../shared/api/resources";
export {
  GetStatefulSetByName,
  GetStatefulSetsSummary,
  GetStatefulSetYAML,
  ListStatefulSets,
  UpdateStatefulSetYAML,
} from "@wailsjs/go/app/App";

export interface StatefulSet {
  Name: string;
  Namespace: string;
  Pods: string;
  Replicas: number;
  Age: string;
  CreatedAt: string;
  Labels: Record<string, string>;
  Annotations: Record<string, string>;
  ManagedFields: ManagedField[];
  Selector: string;
  Images: string[];
  Affinities: number;
  PodStatus: string;
}

export interface StatefulSetSummary {
  Running: number;
  Pending: number;
}

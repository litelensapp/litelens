import type { ManagedField } from "../../../../../shared/api/resources";
import type { TolerationDetail } from "../../pods/api/resources";
export {
  GetDeploymentByName,
  GetDeploymentYAML,
  ListDeployments,
  UpdateDeploymentYAML,
} from "@wailsjs/go/app/App";

export interface DeploymentCondition {
  Type: string;
  Status: string;
  Message?: string;
  Reason?: string;
  LastTransitionTime?: string;
  LastUpdateTime?: string;
}

export interface Deployment {
  Name: string;
  Namespace: string;
  Pods: string;
  Replicas: number;
  Age: string;
  CreatedAt: string;
  Labels: Record<string, string>;
  Annotations: Record<string, string>;
  ManagedFields: ManagedField[];
  ReplicasDetail: string;
  Selector: string;
  NodeSelector: string;
  StrategyType: string;
  Conditions: DeploymentCondition[];
  Tolerations: number;
  TolerationDetails: TolerationDetail[];
  AffinityCount: number;
  Affinities: string;
}

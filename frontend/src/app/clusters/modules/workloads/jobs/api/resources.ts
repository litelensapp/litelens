import type { ManagedField } from "../../../../../shared/api/resources";
export {
  GetJobByName,
  GetJobsSummary,
  GetJobYAML,
  ListJobs,
  UpdateJobYAML,
} from "@wailsjs/go/app/App";

export interface JobCondition {
  Type: string;
  Status: string;
  Message: string;
  Reason: string;
  LastProbeTime: string;
  LastTransitionTime: string;
}

export interface Job {
  Name: string;
  Namespace: string;
  Completions: number;
  Age: string;
  Conditions: JobCondition[];
  Resumed: boolean;
  Status: string;
  Succeeded: number;
  Parallelism: number;
  Duration: string;
  CreatedAt: string;
  Labels: Record<string, string>;
  Annotations: Record<string, string>;
  ManagedFields: ManagedField[];
  Selector: string;
  CompletionMode: string;
  StartTime: string;
  StartTimeAge: string;
  CompletedAt: string;
  CompletedAtAge: string;
  PodsStatuses: string;
  PodStatus: string;
}

export interface JobSummary {
  Succeeded: number;
  Failed: number;
  Pending: number;
}

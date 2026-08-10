export { GetLeaseByName, GetLeaseYAML, ListLeases, UpdateLeaseYAML } from "@wailsjs/go/app/App";
import type { ManagedField } from "../../../../../shared/api/resources";

export interface Lease {
  Name: string;
  Namespace: string;
  HolderIdentity: string;
  LeaseDurationSeconds: number;
  RenewTime: string;
  AcquireTime: string;
  LeaseTransitions: number;
  Age: string;
  CreatedAt: string;
  Labels: Record<string, string>;
  Annotations: Record<string, string>;
  ManagedFields: ManagedField[];
}

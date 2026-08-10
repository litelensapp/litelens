import type { ManagedField } from "../../../../../shared/api/resources";
import type { PolicyRule } from "../../clusterroles/api/resources";
export { GetRoleByName, GetRoleYAML, ListRoles, UpdateRoleYAML } from "@wailsjs/go/app/App";

export interface Role {
  Name: string;
  Namespace: string;
  Age: string;
  CreatedAt: string;
  Labels: Record<string, string>;
  Annotations: Record<string, string>;
  ManagedFields: ManagedField[];
  Rules: PolicyRule[];
}

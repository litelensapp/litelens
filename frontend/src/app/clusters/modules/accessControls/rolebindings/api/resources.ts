import type { ManagedField } from "../../../../../shared/api/resources";
export {
  GetRoleBindingByName,
  GetRoleBindingYAML,
  ListRoleBindings,
  UpdateRoleBindingYAML,
} from "@wailsjs/go/app/App";

export interface RBSubject {
  Kind: string;
  Name: string;
  Namespace: string;
}

export interface RoleBinding {
  Name: string;
  Namespace: string;
  Bindings: string;
  Age: string;
  RoleRefName: string;
  Types: string;
  CreatedAt: string;
  Labels: Record<string, string>;
  Annotations: Record<string, string>;
  ManagedFields: ManagedField[];
  RoleRefKind: string;
  RoleRefGroup: string;
  Subjects: RBSubject[];
}

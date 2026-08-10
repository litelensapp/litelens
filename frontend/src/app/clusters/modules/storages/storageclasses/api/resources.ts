import type { ManagedField } from "../../../../../shared/api/resources";
export {
  GetStorageClassByName,
  GetStorageClassYAML,
  ListStorageClasses,
  UpdateStorageClassYAML,
} from "@wailsjs/go/app/App";

export interface StorageClass {
  Name: string;
  Provisioner: string;
  ReclaimPolicy: string;
  Default: boolean;
  Age: string;
  CreatedAt: string;
  Labels: Record<string, string>;
  Annotations: Record<string, string>;
  ManagedFields: ManagedField[];
  VolumeBindingMode: string;
  MountOptions: string[];
  Parameters: Record<string, string>;
}

export {
  GetConfigMapByName,
  GetConfigMapYAML,
  ListConfigMaps,
  UpdateConfigMapYAML,
} from "@wailsjs/go/app/App";
import type { ManagedField } from "../../../../../shared/api/resources";

export interface ConfigMap {
  Name: string;
  Namespace: string;
  Keys: string[];
  Age: string;
  CreatedAt: string;
  Labels: Record<string, string>;
  Annotations: Record<string, string>;
  ManagedFields: ManagedField[];
  Data: Record<string, string>;
}

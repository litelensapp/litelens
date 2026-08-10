export {
  GetPriorityClassByName,
  GetPriorityClassYAML,
  ListPriorityClasses,
  UpdatePriorityClassYAML,
} from "@wailsjs/go/app/App";

import type { ManagedField } from "../../../../../shared/api/resources";

export interface PriorityClass {
  Name: string;
  Value: number;
  GlobalDefault: boolean;
  Description: string;
  PreemptionPolicy: string;
  Age: string;
  CreatedAt: string;
  ManagedFields: ManagedField[];
}

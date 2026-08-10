import type { SharedNamespaceContext } from "@litelens/design-system";
import type { ManagedField } from "../../../../../shared/api/resources";
export {
  GetNamespaceByName,
  GetNamespaces,
  GetNamespaceYAML,
  ListNamespaces,
  UpdateNamespaceYAML,
} from "@wailsjs/go/app/App";

export interface Namespace extends SharedNamespaceContext {
  Labels: Record<string, string>;
  Annotations: Record<string, string>;
  Age: string;
  CreatedAt: string;
  Status: string;
  ManagedFields: ManagedField[];
  ResourceQuotas: string[];
  LimitRanges: string[];
}

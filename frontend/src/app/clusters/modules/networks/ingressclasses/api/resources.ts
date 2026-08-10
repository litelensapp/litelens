export {
  GetIngressClassByName,
  GetIngressClassYAML,
  ListIngressClasses,
  SetIngressClassAsDefault,
  UnsetIngressClassAsDefault,
  UpdateIngressClassYAML,
} from "@wailsjs/go/app/App";

export interface IngressClass {
  Name: string;
  Controller: string;
  IsDefault: boolean;
  Age: string;
  CreatedAt: string;
  Labels: Record<string, string>;
  Annotations: Record<string, string>;
}

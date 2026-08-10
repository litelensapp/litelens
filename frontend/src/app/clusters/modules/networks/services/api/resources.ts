import type { ManagedField } from "../../../../../shared/api/resources";
export {
  GetServiceByName,
  GetServiceYAML,
  ListServices,
  UpdateServiceYAML,
} from "@wailsjs/go/app/App";

export interface ServicePort {
  Name: string;
  Port: number;
  TargetPort: string;
  Protocol: string;
  NodePort: number;
}

export interface Service {
  Name: string;
  Namespace: string;
  Type: string;
  ClusterIP: string;
  Ports: string;
  ExternalIP: string;
  Selector: string;
  Age: string;
  Status: string;
  // detail fields
  CreatedAt: string;
  Labels: Record<string, string>;
  Annotations: Record<string, string>;
  ManagedFields: ManagedField[];
  SessionAffinity: string;
  InternalTrafficPolicy: string;
  ClusterIPs: string[];
  IPFamilyPolicy: string;
  IPFamilies: string[];
  ServicePorts: ServicePort[];
}

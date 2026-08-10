import type { ManagedField } from "../../../../../shared/api/resources";
export {
  GetEndpointSliceByName,
  GetEndpointSliceYAML,
  ListEndpointSlices,
  UpdateEndpointSliceYAML,
} from "@wailsjs/go/app/App";

export interface EndpointSliceEndpoint {
  Addresses: string[];
  Hostname: string;
  NodeName: string;
  Zone: string;
  TargetName: string;
  TargetKind: string;
  Ready: boolean;
  Serving: boolean;
  Terminating: boolean;
}

export interface EndpointSlicePort {
  Name: string;
  Port: number;
  Protocol: string;
}

export interface EndpointSlice {
  Name: string;
  Namespace: string;
  AddressType: string;
  Age: string;
  Ports: EndpointSlicePort[];
  Endpoints: EndpointSliceEndpoint[];
  // detail fields
  CreatedAt: string;
  ControlledBy: string;
  ServiceName: string;
  Labels: Record<string, string>;
  Annotations: Record<string, string>;
  ManagedFields: ManagedField[];
}

export {
  GetNetworkPolicyByName,
  GetNetworkPolicyYAML,
  ListNetworkPolicies,
  UpdateNetworkPolicyYAML,
} from "@wailsjs/go/app/App";

export type {
  NetworkPolicy,
  NetworkPolicyDetail,
  NetworkPolicyIngressRule,
  NetworkPolicyEgressRule,
  NetworkPolicyPeer,
} from "@litelens/core";

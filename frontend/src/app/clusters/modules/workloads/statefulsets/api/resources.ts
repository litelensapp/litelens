export {
  GetStatefulSetByName,
  GetStatefulSetYAML,
  ListStatefulSets,
  UpdateStatefulSetYAML,
} from "@wailsjs/go/app/App";

export interface StatefulSet {
  Name: string;
  Namespace: string;
  Pods: string;
  Replicas: number;
  Age: string;
  CreatedAt: string;
  Labels: Record<string, string>;
  Annotations: Record<string, string>;
  ManagedFields: string[];
  Selector: string;
  Images: string[];
  Affinities: number;
  PodStatus: string;
}

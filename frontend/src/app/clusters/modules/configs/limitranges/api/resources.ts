export {
  GetLimitRangeByName,
  GetLimitRangeYAML,
  ListLimitRanges,
  UpdateLimitRangeYAML,
} from "@wailsjs/go/app/App";

export interface LimitRange {
  Name: string;
  Namespace: string;
  Age: string;
}

export interface LimitRangeDetail {
  Name: string;
  Namespace: string;
  Age: string;
  CreatedAt: string;
  Labels: Record<string, string>;
  Annotations: Record<string, string>;
  Limits: Record<string, Record<string, Record<string, string>>>;
}

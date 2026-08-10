export {
  GetServiceAccountByName,
  GetServiceAccountYAML,
  ListServiceAccounts,
  UpdateServiceAccountYAML,
} from "@wailsjs/go/app/App";

export interface ServiceAccount {
  Name: string;
  Namespace: string;
  Age: string;
  CreatedAt: string;
  Secrets: string[];
}

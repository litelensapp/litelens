import { Badge } from "@litelens/design-system";
import { FC } from "react";

function statusVariant(status: string) {
  return status === "Terminating" ? "danger" : "success";
}

export const NamespaceStatusBadge: FC<{ status: string }> = ({ status }) => (
  <Badge variant={statusVariant(status)}>{status}</Badge>
);

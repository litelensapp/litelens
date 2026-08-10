import { Badge } from "@litelens/design-system";
import { FC } from "react";

function statusVariant(status: string) {
  return status === "Terminating" ? "danger" : "success";
}

export const ServiceStatusBadge: FC<{ status: string }> = ({ status }) => (
  <Badge variant={statusVariant(status)}>{status}</Badge>
);

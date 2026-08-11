import { Badge } from "@litelens/design-system";
import { FC } from "react";

function conditionVariant(condition: string) {
  switch (condition.toLowerCase()) {
    case "complete":
    case "successcriteriamet":
      return "success";
    case "failed":
    case "failuretarget":
      return "destructive";
    case "suspended":
      return "warning";
    default:
      return "ghost";
  }
}

export const JobConditionBadge: FC<{ condition: string }> = ({ condition }) => (
  <Badge variant={conditionVariant(condition)}>{condition}</Badge>
);

import { FC } from "react";
import { CheckCircle2Icon, Loader2Icon, XCircleIcon } from "../../atoms/icon";

export const LineIcon: FC<{ isError: boolean; isSpinning: boolean }> = ({
  isError,
  isSpinning,
}) => {
  if (isError) return <XCircleIcon className="text-destructive mt-0.5 h-3.5 w-3.5 shrink-0" />;
  if (isSpinning)
    return <Loader2Icon className="text-info mt-0.5 h-3.5 w-3.5 shrink-0 animate-spin" />;
  return <CheckCircle2Icon className="text-success mt-0.5 h-3.5 w-3.5 shrink-0" />;
};

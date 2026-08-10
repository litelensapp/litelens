import { cn } from "../utils/common";
import { FC } from "react";

interface DividerProps {
  className?: string;
}

export const Divider: FC<DividerProps> = ({ className }) => {
  return <hr className={cn("border-border w-full", className)} />;
};

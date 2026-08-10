import { FC } from "react";
import { Button } from "../../atoms/button";
import { PlusIcon } from "../../atoms/icon";
import { Tooltip, TooltipContent, TooltipTrigger } from "../../atoms/tooltip";

interface ResourceCreationButtonProps {
  ariaLabel: string;
  tooltip: string;
  onClick: () => void;
}

export const ResourceCreationButton: FC<ResourceCreationButtonProps> = ({
  ariaLabel,
  tooltip,
  onClick,
}) => (
  <Tooltip>
    <TooltipTrigger>
      <Button size="icon-sm" className="rounded-full" aria-label={ariaLabel} onClick={onClick}>
        <PlusIcon className="size-3.5" />
      </Button>
    </TooltipTrigger>
    <TooltipContent>{tooltip}</TooltipContent>
  </Tooltip>
);

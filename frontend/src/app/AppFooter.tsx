import { InfoIcon, Tooltip, TooltipContent, TooltipTrigger } from "@litelens/design-system";
import { FC } from "react";

interface Props {
  updateInfo: { latestVersion: string; releaseURL: string } | null;
  onUpdateClick: () => void;
}

export const AppFooter: FC<Props> = ({ updateInfo, onUpdateClick }) => (
  <footer className="flex shrink-0 items-center border-t bg-background px-3 py-2">
    {updateInfo && (
      <div className="ml-auto">
        <Tooltip>
          <TooltipTrigger
            onClick={onUpdateClick}
            className="flex items-center gap-1.5 text-xs text-primary underline-offset-2 hover:underline"
          >
            <InfoIcon className="size-3.5" />
            Update available
          </TooltipTrigger>
          <TooltipContent side="top" align="end">
            Version {updateInfo.latestVersion} is available to download
          </TooltipContent>
        </Tooltip>
      </div>
    )}
  </footer>
);

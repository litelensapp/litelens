import { FC } from "react";

interface DownloadProgressIndicatorProps {
  progress?: number; // 0-100 for downloading
  isVerifying?: boolean; // true for verifying state
  isVisible?: boolean; // show/hide
}

export const DownloadProgressIndicator: FC<DownloadProgressIndicatorProps> = ({
  progress = 0,
  isVerifying = false,
  isVisible = false,
}) => {
  if (!isVisible) return null;
  return (
    <div className="w-full">
      <div className="bg-muted h-0.5 overflow-hidden rounded-full">
        {isVerifying ? (
          // Pulse animation while verifying
          <div className="h-full animate-pulse bg-amber-500" />
        ) : (
          // Percentage-driven width while downloading
          <div
            className="h-full bg-amber-500 transition-all duration-300"
            style={{ width: `${Math.min(progress, 100)}%` }}
          />
        )}
      </div>
      {!isVerifying && progress > 0 && (
        <div className="text-muted-foreground mt-1 text-xs">{Math.round(progress)}%</div>
      )}
      {isVerifying && <div className="text-muted-foreground mt-1 text-xs">Verifying...</div>}
    </div>
  );
};

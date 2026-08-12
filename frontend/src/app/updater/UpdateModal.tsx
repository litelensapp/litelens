import {
  ArrowUpCircleIcon,
  Button,
  Dialog,
  DialogContent,
  DialogFooter,
  DialogHeader,
  DialogTitle,
  ExternalLinkIcon,
  Loader2Icon,
} from "@litelens/design-system";
import { FC } from "react";
import { useOpenBrowserURL } from "../shared/hooks/useOpenBrowserURL";
import { usePerformUpdateApp } from "./hooks/data-mutation/usePerformUpdateApp";

interface UpdateModalProps {
  open: boolean;
  onClose: () => void;
  currentVersion: string;
  latestVersion: string;
  releaseURL: string;
  downloadSize: string;
}

export const UpdateModal: FC<UpdateModalProps> = ({
  open,
  onClose,
  currentVersion,
  latestVersion,
  releaseURL,
  downloadSize,
}) => {
  const { mutate: performUpdateApp, isPending: updating, error, reset } = usePerformUpdateApp();
  const openBrowserURL = useOpenBrowserURL();

  function handleClose() {
    reset();
    onClose();
  }

  function handleUpdate() {
    performUpdateApp(latestVersion);
  }

  return (
    <Dialog open={open} onOpenChange={(isOpen) => !isOpen && !updating && handleClose()}>
      <DialogContent showCloseButton={false} className="sm:max-w-sm">
        <DialogHeader>
          <DialogTitle className="flex items-center gap-2">
            <ArrowUpCircleIcon className="text-primary size-4 shrink-0" />
            Update Available
          </DialogTitle>
        </DialogHeader>

        <div className="grid grid-cols-[auto_1fr] gap-x-6 gap-y-2 text-sm">
          <span className="text-muted-foreground text-left">Current version</span>
          <span className="text-right font-mono">{currentVersion}</span>
          <span className="text-muted-foreground text-left">New version</span>
          <span className="text-primary text-right font-mono font-semibold">{latestVersion}</span>
          {Boolean(downloadSize) && (
            <>
              <span className="text-muted-foreground text-left">Download size</span>
              <span className="text-right">{downloadSize}</span>
            </>
          )}
        </div>

        <Button
          variant="link"
          onClick={() => openBrowserURL(releaseURL)}
          className="h-auto w-fit gap-1.5 p-0 text-sm"
        >
          <ExternalLinkIcon className="size-3.5" />
          What&apos;s new
        </Button>

        {error && <p className="text-destructive text-xs">{error.message}</p>}

        <DialogFooter>
          <Button variant="outline" onClick={handleClose} disabled={updating}>
            Cancel
          </Button>
          <Button onClick={handleUpdate} disabled={updating}>
            {updating ? (
              <>
                <Loader2Icon className="size-4 animate-spin" />
                Updating…
              </>
            ) : (
              <>
                <ArrowUpCircleIcon className="size-4" />
                Update Now
              </>
            )}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
};

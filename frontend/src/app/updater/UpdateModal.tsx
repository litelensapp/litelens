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
import { PerformUpdate } from "@wailsjs/go/app/App";
import { BrowserOpenURL } from "@wailsjs/runtime/runtime";
import { FC, useState } from "react";

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
  const [updating, setUpdating] = useState(false);
  const [error, setError] = useState("");

  function handleClose() {
    setError("");
    setUpdating(false);
    onClose();
  }

  async function handleUpdate() {
    setUpdating(true);
    setError("");
    try {
      await PerformUpdate(latestVersion);
    } catch (e) {
      setError(e instanceof Error ? e.message : String(e));
      setUpdating(false);
    }
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
          onClick={() => BrowserOpenURL(releaseURL)}
          className="h-auto w-fit gap-1.5 p-0 text-sm"
        >
          <ExternalLinkIcon className="size-3.5" />
          What&apos;s new
        </Button>

        {error && <p className="text-destructive text-xs">{error}</p>}

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

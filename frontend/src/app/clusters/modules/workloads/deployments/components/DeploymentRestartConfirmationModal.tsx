import { ConfirmationModal } from "@litelens/design-system";
import { FC } from "react";

interface DeploymentRestartConfirmationModalProps {
  open: boolean;
  name: string;
  isPending: boolean;
  onClose: () => void;
  onConfirm: () => void;
}

export const DeploymentRestartConfirmationModal: FC<DeploymentRestartConfirmationModalProps> = ({
  open,
  name,
  isPending,
  onClose,
  onConfirm,
}) => (
  <ConfirmationModal
    open={open}
    title={
      <>
        Restart Deployment:{" "}
        <span className="text-muted-foreground font-mono font-normal">{name}</span>
      </>
    }
    description={
      <>
        This will trigger a rolling restart of{" "}
        <span className="text-foreground font-mono font-medium">{name}</span>. All pods will be
        replaced progressively.
      </>
    }
    confirmLabel="Restart"
    isPending={isPending}
    onClose={onClose}
    onConfirm={onConfirm}
  />
);

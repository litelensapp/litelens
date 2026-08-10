import { ConfirmationModal } from "@litelens/design-system";
import { FC } from "react";

interface NodeCordonConfirmationModalProps {
  open: boolean;
  name: string;
  isPending: boolean;
  onClose: () => void;
  onConfirm: () => void;
}

export const NodeCordonConfirmationModal: FC<NodeCordonConfirmationModalProps> = ({
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
        Cordon Node: <span className="text-muted-foreground font-mono font-normal">{name}</span>
      </>
    }
    description={
      <>
        This will mark <span className="text-foreground font-mono font-medium">{name}</span> as
        unschedulable, preventing new Pods from being scheduled on it. Existing Pods will not be
        evicted.
      </>
    }
    confirmLabel="Cordon"
    confirmVariant="destructive"
    isPending={isPending}
    onClose={onClose}
    onConfirm={onConfirm}
  />
);

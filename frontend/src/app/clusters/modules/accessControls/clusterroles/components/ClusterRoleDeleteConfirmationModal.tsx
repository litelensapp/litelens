import { ConfirmationModal } from "@litelens/design-system";
import { FC, useMemo } from "react";

interface ClusterRoleDeleteConfirmationModalProps {
  open: boolean;
  mode: "single" | "bulk";
  name?: string;
  items?: Array<{ name: string }>;
  isPending: boolean;
  onClose: () => void;
  onConfirm: () => void;
}

export const ClusterRoleDeleteConfirmationModal: FC<ClusterRoleDeleteConfirmationModalProps> = ({
  open,
  mode,
  name,
  items,
  isPending,
  onClose,
  onConfirm,
}) => {
  const { title, description } = useMemo(() => {
    if (mode === "single") {
      return {
        title: (
          <>
            Delete ClusterRole:{" "}
            <span className="text-muted-foreground font-mono font-normal">{name}</span>
          </>
        ),
        description: (
          <>
            This will permanently delete{" "}
            <span className="text-foreground font-mono font-medium">{name}</span>. This action
            cannot be undone.
          </>
        ),
      };
    }

    const count = items?.length ?? 0;

    return {
      title: `Delete ${count} ClusterRole${count === 1 ? "" : "s"}`,
      description: (
        <>
          This will permanently delete {count} clusterrole{count === 1 ? "" : "s"}. This action
          cannot be undone.
        </>
      ),
    };
  }, [mode, name, items]);

  return (
    <ConfirmationModal
      open={open}
      title={title}
      description={description}
      confirmLabel="Delete"
      confirmVariant="destructive"
      isPending={isPending}
      onClose={onClose}
      onConfirm={onConfirm}
    />
  );
};

import { ConfirmationModal } from "@litelens/design-system";
import { FC, useMemo } from "react";

interface StorageClassDeleteConfirmationModalProps {
  open: boolean;
  mode: "single" | "bulk";
  name?: string;
  items?: string[];
  isPending: boolean;
  onClose: () => void;
  onConfirm: () => void;
}

export const StorageClassDeleteConfirmationModal: FC<StorageClassDeleteConfirmationModalProps> = ({
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
            Delete StorageClass:{" "}
            <span className="font-mono font-normal text-muted-foreground">{name}</span>
          </>
        ),
        description: (
          <>
            This will permanently delete{" "}
            <span className="font-mono font-medium text-foreground">{name}</span>. This action
            cannot be undone.
          </>
        ),
      };
    }

    const count = items?.length ?? 0;

    return {
      title: `Delete ${count} StorageClass${count === 1 ? "" : "es"}`,
      description: (
        <>
          This will permanently delete {count} storageclass{count === 1 ? "" : "es"}. This action
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

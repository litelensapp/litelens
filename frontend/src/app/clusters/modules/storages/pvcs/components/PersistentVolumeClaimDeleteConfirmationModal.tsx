import { ConfirmationModal } from "@litelens/design-system";
import { FC, useMemo } from "react";

interface PersistentVolumeClaimDeleteConfirmationModalProps {
  open: boolean;
  mode: "single" | "bulk";
  name?: string;
  namespace?: string;
  items?: Array<{ namespace: string; name: string }>;
  isPending: boolean;
  onClose: () => void;
  onConfirm: () => void;
}

export const PersistentVolumeClaimDeleteConfirmationModal: FC<
  PersistentVolumeClaimDeleteConfirmationModalProps
> = ({ open, mode, name, namespace, items, isPending, onClose, onConfirm }) => {
  const { title, description } = useMemo(() => {
    if (mode === "single") {
      return {
        title: (
          <>
            Delete PersistentVolumeClaim:{" "}
            <span className="text-muted-foreground font-mono font-normal">
              {namespace}/{name}
            </span>
          </>
        ),
        description: (
          <>
            This will permanently delete{" "}
            <span className="text-foreground font-mono font-medium">
              {namespace}/{name}
            </span>
            . This action cannot be undone.
          </>
        ),
      };
    }

    const count = items?.length ?? 0;

    return {
      title: `Delete ${count} PersistentVolumeClaim${count === 1 ? "" : "s"}`,
      description: (
        <>
          This will permanently delete {count} persistentvolumeclaim{count === 1 ? "" : "s"} across{" "}
          {new Set(items?.map((i) => i.namespace) ?? []).size} namespace
          {new Set(items?.map((i) => i.namespace) ?? []).size === 1 ? "" : "s"}. This action cannot
          be undone.
        </>
      ),
    };
  }, [mode, name, namespace, items]);

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

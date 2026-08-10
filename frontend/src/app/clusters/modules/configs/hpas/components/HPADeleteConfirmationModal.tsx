import { ConfirmationModal } from "@litelens/design-system";
import { FC, useMemo } from "react";

interface HPADeleteConfirmationModalProps {
  open: boolean;
  mode: "single" | "bulk";
  name?: string;
  namespace?: string;
  items?: Array<{ namespace: string; name: string }>;
  isPending: boolean;
  onClose: () => void;
  onConfirm: () => void;
}

export const HPADeleteConfirmationModal: FC<HPADeleteConfirmationModalProps> = ({
  open,
  mode,
  name,
  namespace,
  items,
  isPending,
  onClose,
  onConfirm,
}) => {
  const namespaces = useMemo(() => {
    if (mode === "single") return [];
    const set = new Set(items?.map((item) => item.namespace) ?? []);
    return Array.from(set);
  }, [mode, items]);

  const { title, description } = useMemo(() => {
    if (mode === "single") {
      return {
        title: (
          <>
            Delete HPA: <span className="text-muted-foreground font-mono font-normal">{name}</span>
          </>
        ),
        description: (
          <>
            This will permanently delete{" "}
            <span className="text-foreground font-mono font-medium">{name}</span> from namespace{" "}
            <span className="text-foreground font-mono font-medium">{namespace}</span>. This action
            cannot be undone.
          </>
        ),
      };
    }

    const count = items?.length ?? 0;
    const titleText = `Delete ${count} HPA${count === 1 ? "" : "s"}`;

    const descriptionElement = (() => {
      if (namespaces.length === 1) {
        const nsName = namespaces[0];
        return (
          <>
            This will permanently delete {count} hpa{count === 1 ? "" : "s"} from namespace{" "}
            <span className="text-foreground font-mono font-medium">{nsName}</span>. This action
            cannot be undone.
          </>
        );
      }
      return (
        <>
          This will permanently delete {count} hpas from {namespaces.length} namespaces. This action
          cannot be undone.
        </>
      );
    })();

    return {
      title: titleText,
      description: descriptionElement,
    };
  }, [mode, name, namespace, items, namespaces]);

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

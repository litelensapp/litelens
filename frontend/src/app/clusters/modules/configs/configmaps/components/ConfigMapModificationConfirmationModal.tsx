import { ConfirmationModal } from "@litelens/design-system";
import { FC } from "react";

interface ConfigMapModificationConfirmationModalProps {
  open: boolean;
  name: string;
  isPending: boolean;
  editedKeys: string[];
  newEntries: { key: string; value: string }[];
  deletedKeys: string[];
  onClose: () => void;
  onConfirm: () => void;
}

export const ConfigMapModificationConfirmationModal: FC<
  ConfigMapModificationConfirmationModalProps
> = ({ open, name, isPending, editedKeys, newEntries, deletedKeys, onClose, onConfirm }) => {
  const validNew = newEntries.filter((e) => e.key && e.value);

  return (
    <ConfirmationModal
      open={open}
      title={
        <>
          Update ConfigMap:{" "}
          <span className="text-muted-foreground font-mono font-normal">{name}</span>
        </>
      }
      description={
        <div className="flex flex-col gap-3">
          <p>The following changes will be applied to this ConfigMap:</p>
          {editedKeys.length > 0 && (
            <div className="flex flex-col gap-1">
              <p className="text-foreground font-medium">Modified keys</p>
              <ul className="flex flex-col gap-0.5">
                {editedKeys.map((k) => (
                  <li key={k} className="font-mono text-xs">
                    {k}
                  </li>
                ))}
              </ul>
            </div>
          )}
          {validNew.length > 0 && (
            <div className="flex flex-col gap-1">
              <p className="text-foreground font-medium">New keys</p>
              <ul className="flex flex-col gap-0.5">
                {validNew.map((e) => (
                  <li key={e.key} className="font-mono text-xs">
                    {e.key}
                  </li>
                ))}
              </ul>
            </div>
          )}
          {deletedKeys.length > 0 && (
            <div className="flex flex-col gap-1">
              <p className="text-destructive font-medium">Deleted keys</p>
              <ul className="flex flex-col gap-0.5">
                {deletedKeys.map((k) => (
                  <li key={k} className="text-destructive font-mono text-xs line-through">
                    {k}
                  </li>
                ))}
              </ul>
            </div>
          )}
        </div>
      }
      confirmLabel="Update"
      isPending={isPending}
      onClose={onClose}
      onConfirm={onConfirm}
    />
  );
};

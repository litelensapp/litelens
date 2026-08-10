import { FC } from "react";
import { MODIFICATION_TRAY_CONTENT_REGISTRY } from "../../modification/modificationTrayRegistry";
import type { UnifiedTrayContentProps } from "../UnifiedTrayTypes";

export const ModificationTrayFamily: FC<UnifiedTrayContentProps> = ({
  tab,
  collapsed,
  onClose,
}) => {
  if (tab.origin !== "core" || tab.family !== "modification") {
    return null;
  }

  const Content =
    MODIFICATION_TRAY_CONTENT_REGISTRY[tab.kind as keyof typeof MODIFICATION_TRAY_CONTENT_REGISTRY];
  if (!Content) {
    return (
      <div className="text-destructive p-4 text-xs">
        Unknown modification resource kind: {tab.kind}
      </div>
    );
  }

  // eslint-disable-next-line @typescript-eslint/no-explicit-any
  return <Content tab={tab as any} collapsed={collapsed} onClose={onClose} />;
};

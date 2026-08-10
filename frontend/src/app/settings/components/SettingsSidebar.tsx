import {
  AppWindowIcon,
  Button,
  cn,
  HouseIcon,
  LockIcon,
  PackageIcon,
  ServerIcon,
} from "@litelens/design-system";

import { FC } from "react";
import { useIsMarketplaceEnabled } from "../../shared/hooks/useIsMarketplaceEnabled";
import { useIsPrivateRepoAccess } from "../hooks/data-access/useIsPrivateRepoAccess";
import { Section, SECTION_HEADER } from "./types";

interface SettingsSidebarProps {
  section: Section;
  onSelect: (section: Section) => void;
}

export const SettingsSidebar: FC<SettingsSidebarProps> = ({ section, onSelect }) => {
  const { data: isPrivateRepoAccess = true } = useIsPrivateRepoAccess();
  const isMarketplaceEnabled = useIsMarketplaceEnabled();
  return (
    <aside className="flex w-56 shrink-0 flex-col overflow-y-auto border-r">
      <div className="flex flex-col gap-0.5 p-2">
        <Button
          variant="ghost"
          onClick={() => onSelect("welcome")}
          className={cn(
            "w-full justify-start font-medium",
            section === "welcome" && "bg-secondary text-secondary-foreground"
          )}
        >
          <HouseIcon className="size-4 shrink-0" />
          {SECTION_HEADER.welcome}
        </Button>

        <Button
          variant="ghost"
          onClick={() => onSelect("app")}
          className={cn(
            "w-full justify-start font-medium",
            section === "app" && "bg-secondary text-secondary-foreground"
          )}
        >
          <AppWindowIcon className="size-4 shrink-0" />
          {SECTION_HEADER.app}
        </Button>

        <Button
          variant="ghost"
          onClick={() => onSelect("kubernetes")}
          className={cn(
            "w-full justify-start font-medium",
            section === "kubernetes" && "bg-secondary text-secondary-foreground"
          )}
        >
          <ServerIcon className="size-4 shrink-0" />
          {SECTION_HEADER.kubernetes}
        </Button>

        {isMarketplaceEnabled && (
          <Button
            variant="ghost"
            onClick={() => onSelect("marketplace")}
            className={cn(
              "w-full justify-start font-medium",
              section === "marketplace" && "bg-secondary text-secondary-foreground"
            )}
          >
            <PackageIcon className="size-4 shrink-0" />
            {SECTION_HEADER.marketplace}
          </Button>
        )}

        {isPrivateRepoAccess && (
          <Button
            variant="ghost"
            onClick={() => onSelect("sandbox")}
            className={cn(
              "w-full justify-start font-medium",
              section === "sandbox" && "bg-secondary text-secondary-foreground"
            )}
          >
            <LockIcon className="size-4 shrink-0" />
            {SECTION_HEADER.sandbox}
          </Button>
        )}
      </div>
    </aside>
  );
};

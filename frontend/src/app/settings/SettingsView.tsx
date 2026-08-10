import { cn } from "@litelens/design-system";
import { FC, useState } from "react";
import { useIsMarketplaceEnabled } from "../shared/hooks/useIsMarketplaceEnabled";
import { AppContent } from "./components/AppContent";
import { K8sContent } from "./components/K8sContent";
import { MarketplaceContent } from "./components/MarketplaceContent";
import { SandboxContent } from "./components/SandboxContent";
import { SectionHeader } from "./components/SectionHeader";
import { SettingsSidebar } from "./components/SettingsSidebar";
import { Section, SECTION_HEADER } from "./components/types";
import { WelcomeView } from "./components/WelcomeView";
import { useIsPrivateRepoAccess } from "./hooks/data-access/useIsPrivateRepoAccess";

export const SettingsView: FC<{
  initialSection?: Section;
}> = ({ initialSection = "welcome" }) => {
  const { data: isPrivateRepoAccess = true } = useIsPrivateRepoAccess();
  const isMarketplaceEnabled = useIsMarketplaceEnabled();

  const [section, setSection] = useState<Section>(initialSection);

  // Derived every render (not just at mount) so a sandbox selection is never
  // shown once private repo access resolves to disabled, without needing an
  // effect to reset state. Similarly, if marketplace is disabled, redirect to welcome.
  const displaySection =
    (section === "sandbox" && !isPrivateRepoAccess) ||
    (section === "marketplace" && !isMarketplaceEnabled)
      ? "welcome"
      : section;

  return (
    <div className="flex min-h-0 flex-1 overflow-hidden">
      <SettingsSidebar section={displaySection} onSelect={setSection} />

      {/* Content */}
      <div className="flex flex-1 flex-col overflow-hidden">
        {displaySection === "welcome" ? (
          <WelcomeView />
        ) : (
          <>
            <SectionHeader title={SECTION_HEADER[displaySection]} />
            <div
              className={cn(
                "flex flex-1 flex-col px-6 py-4",
                displaySection === "marketplace" ? "overflow-hidden" : "overflow-auto"
              )}
            >
              {displaySection === "sandbox" && <SandboxContent />}
              {displaySection === "kubernetes" && <K8sContent />}
              {displaySection === "app" && <AppContent />}
              {displaySection === "marketplace" && <MarketplaceContent />}
            </div>
          </>
        )}
      </div>
    </div>
  );
};

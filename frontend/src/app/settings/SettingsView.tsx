import { cn } from "@litelens/design-system";
import { FC, useState } from "react";
import { AppContent } from "./components/AppContent";
import { K8sContent } from "./components/K8sContent";
import { MarketplaceContent } from "./components/MarketplaceContent";
import { SandboxContent } from "./components/SandboxContent";
import { SectionHeader } from "./components/SectionHeader";
import { SettingsSidebar } from "./components/SettingsSidebar";
import { Section, SECTION_HEADER } from "./components/types";
import { WelcomeView } from "./components/WelcomeView";

export const SettingsView: FC<{
  initialSection?: Section;
}> = ({ initialSection = "welcome" }) => {
  const [section, setSection] = useState<Section>(initialSection);

  return (
    <div className="flex min-h-0 flex-1 overflow-hidden">
      <SettingsSidebar section={section} onSelect={setSection} />

      {/* Content */}
      <div className="flex flex-1 flex-col overflow-hidden">
        {section === "welcome" ? (
          <WelcomeView />
        ) : (
          <>
            <SectionHeader title={SECTION_HEADER[section]} />
            <div
              className={cn(
                "flex flex-1 flex-col px-6 py-4",
                section === "marketplace" ? "overflow-hidden" : "overflow-auto"
              )}
            >
              {section === "sandbox" && <SandboxContent />}
              {section === "kubernetes" && <K8sContent />}
              {section === "app" && <AppContent />}
              {section === "marketplace" && <MarketplaceContent />}
            </div>
          </>
        )}
      </div>
    </div>
  );
};

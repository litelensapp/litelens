import { useEffect, useState } from "react";
import { EventsOn } from "@wailsjs/runtime/runtime";

export interface ConnectStatusLine {
  message: string;
  isError: boolean;
}

export interface WarningBanner {
  title: string;
  description?: string;
}

export interface ConnectStatusResult {
  lines: ConnectStatusLine[];
  warningBanner: WarningBanner | null;
}

export function useConnectStatusEvents(contextName: string): ConnectStatusResult {
  const [lines, setLines] = useState<ConnectStatusLine[]>([]);
  const [warningBanner, setWarningBanner] = useState<WarningBanner | null>(null);

  useEffect(() => {
    let isActive = true;

    const unsubscribeStatus = EventsOn(
      "connect:status",
      (payload: { context: string; message: string }) => {
        if (payload.context !== contextName || !isActive) return;
        const setupScriptMessages = [
          "Starting proxy setup script...",
          "Waiting for setup script...",
        ];
        const isSetupMessage = setupScriptMessages.some((msg) => payload.message.includes(msg));
        const isError =
          !isSetupMessage &&
          (payload.message.startsWith("Failed") || payload.message.startsWith("Cannot"));
        setLines((prev) => [...prev, { message: payload.message, isError }]);
      }
    );

    const unsubscribeDegraded = EventsOn(
      "SetupScriptDegraded",
      (payload: { context: string; message: string }) => {
        if (payload.context !== contextName || !isActive) return;
        setWarningBanner({ title: "Setup Script Failed", description: payload.message });
      }
    );

    const unsubscribeCrashed = EventsOn(
      "SetupScriptCrashed",
      (payload: { context: string; message: string }) => {
        if (payload.context !== contextName || !isActive) return;
        setWarningBanner({ title: "Setup Script Crashed", description: payload.message });
      }
    );

    const unsubscribePortInUse = EventsOn(
      "SetupScriptPortInUse",
      (payload: { context: string; message: string }) => {
        if (payload.context !== contextName || !isActive) return;
        setWarningBanner({ title: "Setup Script Error", description: payload.message });
      }
    );

    return () => {
      isActive = false;
      unsubscribeStatus();
      unsubscribeDegraded();
      unsubscribeCrashed();
      unsubscribePortInUse();
    };
  }, [contextName]);

  return { lines, warningBanner };
}

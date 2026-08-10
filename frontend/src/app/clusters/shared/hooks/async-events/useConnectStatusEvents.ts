import { useEffect, useState } from "react";
import { EventsOn } from "@wailsjs/runtime/runtime";

export interface ConnectStatusLine {
  message: string;
  isError: boolean;
}

export function useConnectStatusEvents(contextName: string): ConnectStatusLine[] {
  const [lines, setLines] = useState<ConnectStatusLine[]>([]);

  useEffect(() => {
    return EventsOn("connect:status", (payload: { context: string; message: string }) => {
      if (payload.context !== contextName) return;
      const isError = payload.message.startsWith("Failed") || payload.message.startsWith("Cannot");
      setLines((prev) => [...prev, { message: payload.message, isError }]);
    });
  }, [contextName]);

  return lines;
}

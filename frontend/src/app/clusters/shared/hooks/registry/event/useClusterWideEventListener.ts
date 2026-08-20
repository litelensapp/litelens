import { useEffect } from "react";
import { EventsOn } from "@wailsjs/runtime/runtime";
import { clearRegistry, getHandler } from "./clusterWideEventRegistry";

interface PluginEventPayload {
  pluginId: string;
  eventName: string;
  payload: unknown;
}

export function useClusterWideEventListener(): void {
  useEffect(() => {
    clearRegistry();
    return EventsOn("plugin:event", (eventPayload: unknown) => {
      const payload = eventPayload as PluginEventPayload;
      const handler = getHandler(payload.eventName);
      if (handler) {
        try {
          handler(payload.payload);
        } catch (err) {
          console.error(`Error in handler for event ${payload.eventName}:`, err);
        }
      }
    });
  }, []);
}

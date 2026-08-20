import { useEffect } from "react";
import { registerHandler } from "./clusterWideEventRegistry";

export function useRegisterClusterWideEvents(
  handlers: Record<string, (payload: unknown) => void>
): void {
  useEffect(() => {
    for (const [eventName, handler] of Object.entries(handlers)) {
      registerHandler(eventName, handler);
    }
    // No cleanup — handlers must persist after unmount (see bugfix note above).
  }, [handlers]);
}

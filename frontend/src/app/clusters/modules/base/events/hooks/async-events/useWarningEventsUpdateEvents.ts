import { startTransition, useEffect, useState } from "react";
import { EventsOn } from "@wailsjs/runtime/runtime";

export function useWarningEventsUpdateEvents(namespaces: string[] = []): boolean {
  const [triggerRefresh, setTriggerRefresh] = useState(false);
  const [prevNamespaces, setPrevNamespaces] = useState(namespaces);

  // When namespace selection changes, reset the trigger flag.
  if (JSON.stringify(prevNamespaces) !== JSON.stringify(namespaces)) {
    setPrevNamespaces(namespaces);
    setTriggerRefresh(false);
  }

  useEffect(() => {
    if (namespaces.length === 0) {
      return EventsOn("events:warning:update", () => {
        startTransition(() => {
          setTriggerRefresh((prev) => !prev);
        });
      });
    }

    const unsubscribers: Array<() => void> = [];
    for (const ns of namespaces) {
      const eventName = `events:${ns}:warning:update`;
      const unsubscriber = EventsOn(eventName, () => {
        startTransition(() => {
          setTriggerRefresh((prev) => !prev);
        });
      });
      unsubscribers.push(unsubscriber);
    }

    return () => {
      unsubscribers.forEach((unsub) => unsub());
    };
  }, [namespaces]);
  return triggerRefresh;
}

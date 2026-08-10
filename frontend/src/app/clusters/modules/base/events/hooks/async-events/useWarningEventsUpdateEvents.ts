import { useEffect, useState } from "react";
import { EventsOn } from "@wailsjs/runtime/runtime";

export function useWarningEventsUpdateEvents(namespace = ""): boolean {
  const [triggerRefresh, setTriggerRefresh] = useState(false);
  const [prevNamespace, setPrevNamespace] = useState(namespace);

  // Reset stale data from the previous namespace's channel before re-subscribing.
  if (namespace !== prevNamespace) {
    setPrevNamespace(namespace);
    setTriggerRefresh(false);
  }

  useEffect(() => {
    const eventName = namespace ? `events:${namespace}:warning:update` : "events:warning:update";
    return EventsOn(eventName, () => setTriggerRefresh((prev) => !prev));
  }, [namespace]);
  return triggerRefresh;
}

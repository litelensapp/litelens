import { startTransition, useEffect, useState } from "react";
import { EventsOn } from "@wailsjs/runtime/runtime";
import type { Event } from "../../api/resources";

// Data-only event hook: tracks the latest pushed events in local state.
// Called directly from event data-access hooks (useGetEvents, useGetEventDetail)
// to merge event-driven data locally without cache-wide side effects.
// Events have no YAML view (raw Kubernetes Events aren't edited), so no YAML hook to wire.
export function useEventsUpdateEvents(namespace = ""): Event[] {
  const [latestEvents, setLatestEvents] = useState<Event[]>([]);
  const [prevNamespace, setPrevNamespace] = useState(namespace);

  // Reset stale data from the previous namespace's channel before re-subscribing.
  if (namespace !== prevNamespace) {
    setPrevNamespace(namespace);
    setLatestEvents([]);
  }

  useEffect(() => {
    const eventName = namespace ? `events:${namespace}:update` : "events:update";
    return EventsOn(eventName, (data: Event[]) => {
      startTransition(() => {
        setLatestEvents(data);
      });
    });
  }, [namespace]);
  return latestEvents;
}

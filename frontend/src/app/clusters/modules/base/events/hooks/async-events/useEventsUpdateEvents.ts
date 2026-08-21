import { startTransition, useEffect, useState } from "react";
import { EventsOn } from "@wailsjs/runtime/runtime";
import type { Event } from "../../api/resources";
import { mergeNamespaceScopedData } from "../../../../../shared/utils/eventMerging";

// Data-only event hook: tracks the latest pushed events in local state.
// Called directly from event data-access hooks (useGetEvents, useGetEventDetail)
// to merge event-driven data locally without cache-wide side effects.
// Events have no YAML view (raw Kubernetes Events aren't edited), so no YAML hook to wire.
export function useEventsUpdateEvents(namespaces: string[] = []): Event[] {
  const [latestEvents, setLatestEvents] = useState<Event[]>([]);
  const [prevNamespaces, setPrevNamespaces] = useState(namespaces);

  // When namespace selection changes, filter down accumulated state to only selected namespaces.
  if (JSON.stringify(prevNamespaces) !== JSON.stringify(namespaces)) {
    setPrevNamespaces(namespaces);
    if (namespaces.length > 0) {
      const namespacesSet = new Set(namespaces);
      setLatestEvents((prev) => prev.filter((event) => namespacesSet.has(event.Namespace)));
    } else {
      setLatestEvents([]);
    }
  }

  useEffect(() => {
    if (namespaces.length === 0) {
      return EventsOn("events:update", (data: Event[]) => {
        startTransition(() => {
          setLatestEvents(data);
        });
      });
    }

    const unsubscribers: Array<() => void> = [];
    for (const ns of namespaces) {
      const eventName = `events:${ns}:update`;
      const unsubscriber = EventsOn(eventName, (data: Event[]) => {
        startTransition(() => {
          setLatestEvents((prev) => mergeNamespaceScopedData(prev, data, ns));
        });
      });
      unsubscribers.push(unsubscriber);
    }

    return () => {
      unsubscribers.forEach((unsub) => unsub());
    };
  }, [namespaces]);
  return latestEvents;
}

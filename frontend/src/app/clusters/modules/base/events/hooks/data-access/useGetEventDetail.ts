import { useQuery } from "@tanstack/react-query";
import { useMemo } from "react";
import { DEFAULT_QUERY_OPTIONS } from "../../../../../../shared/api/api";
import { QUERY_KEY_EVENT_DETAIL } from "../../api/api.const";
import type { Event } from "../../api/resources";
import { GetEventByName } from "../../api/resources";
import { useEventsUpdateEvents } from "../async-events/useEventsUpdateEvents";

export const useGetEventDetail = (context: string, namespace: string, name: string) => {
  const latestEvents = useEventsUpdateEvents([namespace]);

  const query = useQuery<Event, Error>({
    queryKey: [QUERY_KEY_EVENT_DETAIL, { context, namespace, name }],
    queryFn: () => GetEventByName(namespace, name),
    ...DEFAULT_QUERY_OPTIONS,
    enabled: !!context && !!namespace && !!name,
  });

  // Merge event-driven data: prefer matched event from latest event if available.
  const mergedData = useMemo(() => {
    const matchedEvent = latestEvents.find((e) => e.Namespace === namespace && e.Name === name);
    if (matchedEvent) return matchedEvent;
    return query.data;
  }, [latestEvents, query.data, namespace, name]);

  return {
    ...query,
    data: mergedData,
  };
};

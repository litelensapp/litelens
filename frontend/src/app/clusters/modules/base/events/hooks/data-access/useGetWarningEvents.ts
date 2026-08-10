import { useQuery } from "@tanstack/react-query";
import { DEFAULT_QUERY_OPTIONS } from "../../../../../../shared/api/api";
import { QUERY_KEY_WARNING_EVENTS } from "../../api/api.const";
import type { Event } from "../../api/resources";
import { ListWarningEvents } from "../../api/resources";
import { useWarningEventsUpdateEvents } from "../async-events/useWarningEventsUpdateEvents";

export const useGetWarningEvents = (input: { context: string; namespace: string }) => {
  const { context, namespace } = input;
  const triggerRefresh = useWarningEventsUpdateEvents(namespace);

  return useQuery<Event[], Error>({
    queryKey: [QUERY_KEY_WARNING_EVENTS, { context, namespace }, triggerRefresh],
    queryFn: () =>
      ListWarningEvents(namespace).then((events) =>
        events.toSorted((a, b) => b.CreatedAt - a.CreatedAt)
      ),
    ...DEFAULT_QUERY_OPTIONS,
    enabled: !!context,
  });
};

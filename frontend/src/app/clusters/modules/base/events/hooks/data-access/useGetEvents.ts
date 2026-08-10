import type { UseQueryCallback } from "@litelens/design-system";
import { useQuery } from "@tanstack/react-query";
import { useMemo } from "react";
import { DEFAULT_QUERY_OPTIONS } from "../../../../../../shared/api/api";
import { QUERY_KEY_EVENTS } from "../../api/api.const";
import type { Event } from "../../api/resources";
import { ListEvents } from "../../api/resources";
import { useEventsUpdateEvents } from "../async-events/useEventsUpdateEvents";

export const useGetEvents = (
  input: { context: string; namespace: string },
  callback?: UseQueryCallback<Event[]>
) => {
  const { context, namespace } = input;
  const latestEvents = useEventsUpdateEvents(namespace);

  const query = useQuery<Event[], Error>({
    queryKey: [QUERY_KEY_EVENTS, { context, namespace }],
    queryFn: () => ListEvents(namespace),
    ...DEFAULT_QUERY_OPTIONS,
    enabled: !!context,
  });

  // Merge event-driven data locally: prefer event-filtered events over fetched data if available.
  // Filter cluster-wide event list to this hook's namespace (or include all if namespace === "").
  const mergedData = useMemo(() => {
    let baseData = query.data;
    if (latestEvents.length)
      baseData =
        namespace === ""
          ? latestEvents
          : latestEvents.filter((event) => event.Namespace === namespace);
    return callback?.select ? callback.select(baseData) : baseData;
  }, [latestEvents, query.data, namespace, callback]);

  return {
    ...query,
    data: mergedData,
  };
};

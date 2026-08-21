import { DEFAULT_QUERY_OPTIONS } from "../../../../../../shared/api/api";
import type { UseQueryCallback } from "@litelens/core";
import { useMemo } from "react";
import { useQuery } from "@tanstack/react-query";
import { QUERY_KEY_EVENTS } from "../../api/api.const";
import type { Event } from "../../api/resources";
import { ListEvents } from "../../api/resources";
import { filterByNamespaces } from "../../../../../shared/utils/namespaceFiltering";
import { useEventsUpdateEvents } from "../async-events/useEventsUpdateEvents";

export const useGetEvents = (
  input: { context: string; namespaces: string[] },
  callback?: UseQueryCallback<Event[]>
) => {
  const { context, namespaces } = input;
  const latestEvents = useEventsUpdateEvents(namespaces);

  const query = useQuery<Event[], Error>({
    queryKey: [QUERY_KEY_EVENTS, { context, namespaces }],
    queryFn: () => ListEvents(namespaces),
    ...DEFAULT_QUERY_OPTIONS,
    enabled: !!context,
  });

  const mergedData = useMemo(() => {
    let baseData = query.data;
    if (latestEvents.length) baseData = filterByNamespaces(latestEvents, namespaces);
    return callback?.select ? callback.select(baseData) : baseData;
  }, [latestEvents, query.data, namespaces, callback]);

  return { ...query, data: mergedData };
};

import { DEFAULT_QUERY_OPTIONS } from "../../../../../../shared/api/api";
import type { UseQueryCallback } from "@litelens/design-system";
import { useQuery } from "@tanstack/react-query";
import { useMemo } from "react";
import { QUERY_KEY_SERVICES } from "../../api/api.const";
import type { Service } from "../../api/resources";
import { ListServices } from "../../api/resources";
import { useServicesUpdateEvents } from "../async-events/useServicesUpdateEvents";

export const useGetServices = (
  input: { context: string; namespace: string },
  callback?: UseQueryCallback<Service[]>
) => {
  const { context, namespace } = input;
  const latestServices = useServicesUpdateEvents();

  const query = useQuery<Service[], Error>({
    queryKey: [QUERY_KEY_SERVICES, { context, namespace }],
    queryFn: () => ListServices(namespace),
    ...DEFAULT_QUERY_OPTIONS,
    enabled: !!context,
  });

  // Merge event-driven data locally: prefer event-filtered services over fetched data if available.
  // Filter cluster-wide event list to this hook's namespace (or include all if namespace === "").
  const mergedData = useMemo(() => {
    let baseData = query.data;
    if (latestServices.length)
      baseData =
        namespace === ""
          ? latestServices
          : latestServices.filter((svc) => svc.Namespace === namespace);
    return callback?.select ? callback.select(baseData) : baseData;
  }, [latestServices, query.data, namespace, callback]);

  return {
    ...query,
    data: mergedData,
  };
};

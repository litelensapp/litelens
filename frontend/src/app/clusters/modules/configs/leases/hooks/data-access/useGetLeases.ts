import { DEFAULT_QUERY_OPTIONS } from "../../../../../../shared/api/api";
import type { UseQueryCallback } from "@litelens/design-system";
import { useQuery } from "@tanstack/react-query";
import { useMemo } from "react";
import { QUERY_KEY_LEASES } from "../../api/api.const";
import type { Lease } from "../../api/resources";
import { ListLeases } from "../../api/resources";
import { useLeasesUpdateEvents } from "../async-events/useLeasesUpdateEvents";

export const useGetLeases = (
  input: { context: string; namespace: string },
  callback?: UseQueryCallback<Lease[]>
) => {
  const { context, namespace } = input;
  const latestLeases = useLeasesUpdateEvents(namespace);

  const query = useQuery<Lease[], Error>({
    queryKey: [QUERY_KEY_LEASES, { context, namespace }],
    queryFn: () => ListLeases(namespace),
    ...DEFAULT_QUERY_OPTIONS,
    enabled: !!context,
  });

  const mergedData = useMemo(() => {
    let baseData = query.data;
    if (latestLeases.length)
      baseData =
        namespace === ""
          ? latestLeases
          : latestLeases.filter((lease) => lease.Namespace === namespace);
    return callback?.select ? callback.select(baseData) : baseData;
  }, [latestLeases, query.data, namespace, callback]);

  return {
    ...query,
    data: mergedData,
  };
};

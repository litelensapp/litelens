import { DEFAULT_QUERY_OPTIONS } from "../../../../../../shared/api/api";
import { QUERY_KEY_LEASE_YAML } from "../../api/api.const";
import { GetLeaseYAML } from "../../api/resources";
import { useQuery, useQueryClient } from "@tanstack/react-query";
import { useEffect, useMemo } from "react";
import { useLeasesUpdateEvents } from "../async-events/useLeasesUpdateEvents";

export function useGetLeaseYAML(context: string, namespace: string, name: string, enabled = true) {
  const queryClient = useQueryClient();
  const latestLeases = useLeasesUpdateEvents(namespace);

  const query = useQuery({
    queryKey: [QUERY_KEY_LEASE_YAML, { context, namespace, name }],
    queryFn: () => GetLeaseYAML(namespace, name),
    ...DEFAULT_QUERY_OPTIONS,
    enabled: !!context && !!namespace && !!name && enabled,
  });

  const leaseKeyDependency = useMemo(() => {
    const matchedLease = latestLeases.find((l) => l.Namespace === namespace && l.Name === name);
    if (matchedLease) return JSON.stringify(matchedLease);
    return null;
  }, [latestLeases, namespace, name]);

  useEffect(() => {
    if (leaseKeyDependency)
      queryClient.invalidateQueries({
        queryKey: [QUERY_KEY_LEASE_YAML, { context, namespace, name }],
      });
  }, [leaseKeyDependency, context, namespace, name, queryClient]);

  return query;
}

import { useQuery, useQueryClient } from "@tanstack/react-query";
import { useEffect, useMemo } from "react";
import { QUERY_KEY_INGRESS_CLASS_YAML } from "../../api/api.const";
import { DEFAULT_QUERY_OPTIONS } from "../../../../../../shared/api/api";
import { GetIngressClassYAML } from "../../api/resources";
import { useIngressClassesUpdateEvents } from "../async-events/useIngressClassesUpdateEvents";

export function useGetIngressClassYAML(context: string, name: string, enabled = true) {
  const latestIngressClasses = useIngressClassesUpdateEvents();
  const queryClient = useQueryClient();
  const query = useQuery({
    queryKey: [QUERY_KEY_INGRESS_CLASS_YAML, { context, name }],
    queryFn: () => GetIngressClassYAML(name),
    ...DEFAULT_QUERY_OPTIONS,
    enabled: !!context && !!name && enabled,
  });

  const matchedIngressClass = useMemo(
    () => latestIngressClasses.find((ic) => ic.Name === name),
    [latestIngressClasses, name]
  );
  const matchedIngressClassKey = JSON.stringify(matchedIngressClass);

  useEffect(() => {
    if (matchedIngressClass) {
      queryClient.invalidateQueries({
        queryKey: [QUERY_KEY_INGRESS_CLASS_YAML, { context, name }],
      });
    }
  }, [matchedIngressClass, matchedIngressClassKey, context, name, queryClient]);

  return query;
}

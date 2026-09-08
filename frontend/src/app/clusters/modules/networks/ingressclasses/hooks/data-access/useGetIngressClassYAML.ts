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

  const ingressClassKeyDependency = useMemo(() => {
    const matchedIngressClass = latestIngressClasses.find((ic) => ic.Name === name);
    if (matchedIngressClass) return JSON.stringify(matchedIngressClass);
    return null;
  }, [latestIngressClasses, name]);

  useEffect(() => {
    if (ingressClassKeyDependency) {
      queryClient.invalidateQueries({
        queryKey: [QUERY_KEY_INGRESS_CLASS_YAML, { context, name }],
      });
    }
  }, [ingressClassKeyDependency, context, name, queryClient]);

  return query;
}

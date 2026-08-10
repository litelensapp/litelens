import { useQuery } from "@tanstack/react-query";
import { useMemo } from "react";
import { QUERY_KEY_INGRESSCLASS_DETAIL } from "../../api/api.const";
import { DEFAULT_QUERY_OPTIONS } from "../../../../../../shared/api/api";
import type { IngressClass } from "../../api/resources";
import { GetIngressClassByName } from "../../api/resources";
import { useIngressClassesUpdateEvents } from "../async-events/useIngressClassesUpdateEvents";

export const useGetIngressClassDetail = (context: string, name: string) => {
  const latestIngressClasses = useIngressClassesUpdateEvents();
  const query = useQuery<IngressClass, Error>({
    queryKey: [QUERY_KEY_INGRESSCLASS_DETAIL, { context, name }],
    queryFn: () => GetIngressClassByName(name),
    ...DEFAULT_QUERY_OPTIONS,
    enabled: !!context && !!name,
  });

  const mergedData = useMemo(() => {
    const matchedIngressClass = latestIngressClasses.find((ic) => ic.Name === name);
    if (matchedIngressClass) return matchedIngressClass;
    return query.data;
  }, [latestIngressClasses, query.data, name]);

  return { ...query, data: mergedData };
};

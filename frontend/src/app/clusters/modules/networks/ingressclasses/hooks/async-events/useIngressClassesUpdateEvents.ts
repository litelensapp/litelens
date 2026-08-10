import { useEffect, useState } from "react";
import { EventsOn } from "@wailsjs/runtime/runtime";
import type { IngressClass } from "../../api/resources";

export function useIngressClassesUpdateEvents(): IngressClass[] {
  const [latestIngressClasses, setLatestIngressClasses] = useState<IngressClass[]>([]);
  useEffect(() => {
    return EventsOn("ingressclasses:update", (data: IngressClass[]) =>
      setLatestIngressClasses(data)
    );
  }, []);
  return latestIngressClasses;
}

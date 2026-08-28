import { renderErrorToast } from "@litelens/design-system";
import { useEffect, useRef, useState } from "react";
import { EventsOn } from "@wailsjs/runtime/runtime";
import { IsResourceForbidden } from "@wailsjs/go/app/App";

interface UseCatchForbiddenResourceOptions {
  open?: boolean;
  resourceName?: string | null;
  resourceLabel?: string | null;
  onForbiddenDetected?: () => void;
  labelMap?: Record<string, string>;
  activeContext?: string;
}

interface UseCatchForbiddenResourceResult {
  forbiddenResources: Set<string>;
}

export const useCatchForbiddenResource = (
  activeResource: string,
  options?: UseCatchForbiddenResourceOptions
): UseCatchForbiddenResourceResult => {
  const [forbiddenResources, setForbiddenResources] = useState<Set<string>>(new Set());

  // Reset forbidden state when the drawer transitions from closed to open, or when
  // the active cluster context changes. Using the React-approved derived-state pattern
  // (setState during render with a prev-value guard) so the reset is synchronous,
  // before paint, with no extra effect.
  const [prevOpen, setPrevOpen] = useState(options?.open);
  if (options?.open !== prevOpen) {
    setPrevOpen(options?.open);
    if (options?.open) setForbiddenResources(new Set());
  }
  const [prevActiveContext, setPrevActiveContext] = useState(options?.activeContext);
  if (options?.activeContext !== prevActiveContext) {
    setPrevActiveContext(options?.activeContext);
    setForbiddenResources(new Set());
  }

  // Tracks whether a toast was already shown in the current open session to prevent
  // a double-toast race between the live Wails event path and the poll below.
  const drawerToastFiredRef = useRef(false);

  // Keep a single ref current so the Wails event handler always reads the latest values
  // without being re-subscribed on every render.
  const activeResourceRef = useRef(activeResource);
  const optionsRef = useRef(options);
  useEffect(() => {
    activeResourceRef.current = activeResource;
    optionsRef.current = options;
  });

  useEffect(() => {
    const unsub = EventsOn("resource:forbidden", (resource: string) => {
      setForbiddenResources((prev) => new Set([...prev, resource]));
      if (activeResourceRef.current !== resource) return;

      const opts = optionsRef.current;
      if (opts?.resourceName && opts?.open) {
        // Drawer mode: show per-resource "cannot get" toast and close the drawer
        drawerToastFiredRef.current = true;
        const label = opts.resourceLabel ?? activeResourceRef.current;
        renderErrorToast({ title: `Access denied: cannot get ${label} "${opts.resourceName}"` });
        opts.onForbiddenDetected?.();
      } else if (opts?.labelMap) {
        // List-view mode: show "cannot list X" toast using labelMap
        const label = opts.labelMap[resource] ?? resource;
        renderErrorToast({ title: `Access denied: cannot list ${label}` });
      }
    });
    return () => {
      if (typeof unsub === "function") unsub();
    };
  }, []);

  // On every open transition, check whether the resource is already forbidden in
  // the informer cache and show the drawer toast immediately (handles the case
  // where the 403 was received before the drawer was opened and won't be re-emitted).
  const open = options?.open;
  useEffect(() => {
    let cancelled = false;
    if (!open) {
      drawerToastFiredRef.current = false;
      return;
    }
    // Drawer mode only — no labelMap.
    const opts = optionsRef.current;
    if (!opts?.resourceName) return;

    IsResourceForbidden(activeResourceRef.current).then((forbidden) => {
      if (cancelled || !forbidden || drawerToastFiredRef.current) return;
      const currentOpts = optionsRef.current;
      if (!currentOpts?.resourceName) return;
      drawerToastFiredRef.current = true;
      const label = currentOpts.resourceLabel ?? activeResourceRef.current;
      renderErrorToast({
        title: `Access denied: cannot get ${label} "${currentOpts.resourceName}"`,
      });
      currentOpts.onForbiddenDetected?.();
    });
    return () => {
      cancelled = true;
    };
  }, [open]);

  return { forbiddenResources };
};

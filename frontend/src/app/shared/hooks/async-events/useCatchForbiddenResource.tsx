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
  // Namespace of the specific resource this hook instance cares about (drawer
  // mode). A forbidden event/poll result for a different namespace is ignored,
  // so denying access in namespace B doesn't toast/close a drawer open on a
  // resource in namespace A. Omit for cluster-scoped resources.
  namespace?: string;
}

interface UseCatchForbiddenResourceResult {
  forbiddenResources: Set<string>;
}

// Renders as "" for a cluster-wide denial (namespace === "") so toast copy doesn't
// claim a specific namespace when none was actually identified.
const namespaceSuffix = (namespace: string): string =>
  namespace ? ` in namespace "${namespace}"` : "";

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
    const unsub = EventsOn(
      "resource:forbidden",
      (payload: { resource: string; namespace: string }) => {
        const { resource, namespace } = payload;
        setForbiddenResources((prev) => new Set([...prev, resource]));
        if (activeResourceRef.current !== resource) return;

        const opts = optionsRef.current;
        // A namespace-scoped forbidden event only applies to this hook instance
        // when it's cluster-wide (namespace === "") or matches the specific
        // namespace this instance cares about.
        const appliesToThisNamespace =
          namespace === "" || !opts?.namespace || opts.namespace === namespace;
        if (!appliesToThisNamespace) return;

        if (opts?.resourceName && opts?.open) {
          // Drawer mode: show per-resource "cannot get" toast and close the drawer
          drawerToastFiredRef.current = true;
          const label = opts.resourceLabel ?? activeResourceRef.current;
          renderErrorToast({
            title: `Access denied: cannot get ${label} "${opts.resourceName}"${namespaceSuffix(namespace)}`,
          });
          opts.onForbiddenDetected?.();
        } else if (opts?.labelMap) {
          // List-view mode: show "cannot list X" toast using labelMap
          const label = opts.labelMap[resource] ?? resource;
          renderErrorToast({
            title: `Access denied: cannot list ${label}${namespaceSuffix(namespace)}`,
          });
        }
      }
    );
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

    IsResourceForbidden(activeResourceRef.current, opts.namespace ?? "").then((forbidden) => {
      if (cancelled || !forbidden || drawerToastFiredRef.current) return;
      const currentOpts = optionsRef.current;
      if (!currentOpts?.resourceName) return;
      drawerToastFiredRef.current = true;
      const label = currentOpts.resourceLabel ?? activeResourceRef.current;
      renderErrorToast({
        title: `Access denied: cannot get ${label} "${currentOpts.resourceName}"${namespaceSuffix(currentOpts.namespace ?? "")}`,
      });
      currentOpts.onForbiddenDetected?.();
    });
    return () => {
      cancelled = true;
    };
  }, [open]);

  // List-view mode reconciliation: the live "resource:forbidden" event fires at
  // most once per resource for the lifetime of the backend's cluster connection
  // (see FactoryHandle.StopResource), but a frontend-only reload (e.g. Ctrl+R,
  // which remounts React without restarting the Go backend) resets
  // forbiddenResources to empty without a new event ever coming. Without this,
  // a resource forbidden before the reload would silently stop toasting/graying
  // out in the nav until the next full cluster reconnect. Poll each labelMap
  // resource's already-known state once on mount / activeContext change so a
  // pre-existing 403 is still reflected after a reload.
  const activeContext = options?.activeContext;
  useEffect(() => {
    const labelMap = optionsRef.current?.labelMap;
    if (!labelMap) return;
    let cancelled = false;
    Promise.all(
      Object.keys(labelMap).map((resource) =>
        IsResourceForbidden(resource, "").then((forbidden) => (forbidden ? resource : null))
      )
    ).then((results) => {
      if (cancelled) return;
      const forbidden = results.filter((r): r is string => r !== null);
      if (forbidden.length > 0) {
        setForbiddenResources((prev) => new Set([...prev, ...forbidden]));
      }
    });
    return () => {
      cancelled = true;
    };
  }, [activeContext]);

  return { forbiddenResources };
};

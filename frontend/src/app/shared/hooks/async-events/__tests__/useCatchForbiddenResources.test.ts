/**
 * Tests for useCatchForbiddenResources
 *
 * Bug fixed: the list-view branch was matching closed-drawer callers that pass
 * no labelMap (resourceName=null, no opts). The fix guards the branch with
 * `opts?.labelMap &&` so only explicit list-view callers enter it.
 *
 * Scenarios verified:
 *   A - Closed drawer (open=false, resourceName=null, no labelMap) → no toast
 *   B - Open drawer (open=true, resourceName="my-node") → toast + onForbiddenDetected
 *   C - List-view caller (labelMap present, no resourceName) → human-readable toast
 *   D - Event for a different resource → hook ignores it (no toast, no callback)
 *   E - forbiddenResources set is always updated regardless of branch
 */

import { vi, describe, it, expect, beforeEach } from "vitest";
import { renderHook, act } from "@testing-library/react";
import { useCatchForbiddenResources } from "../useCatchForbiddenResources";

// ─── Wails runtime mock ───────────────────────────────────────────────────────
// vi.hoisted runs before vi.mock hoisting, letting us share state between the
// mock factory and the test body.
type EventHandler = (resource: string) => void;

const { getHandler, setHandler } = vi.hoisted(() => {
  let _handler: EventHandler | null = null;
  return {
    getHandler: () => _handler,
    setHandler: (h: EventHandler | null) => {
      _handler = h;
    },
  };
});

vi.mock("@wailsjs/runtime/runtime", () => ({
  EventsOn: vi.fn((_eventName: string, handler: EventHandler) => {
    setHandler(handler);
    return () => setHandler(null);
  }),
}));

// ─── Wails Go binding mock ────────────────────────────────────────────────────
const isResourceForbiddenMock = vi.hoisted(() => vi.fn().mockResolvedValue(false));
vi.mock("@wailsjs/go/app/App", () => ({
  IsResourceForbidden: isResourceForbiddenMock,
}));

// ─── Toast mocks ──────────────────────────────────────────────────────────────
const ErrorToastSpy = vi.hoisted(() => vi.fn(({ title }: { title: string }) => title));
vi.mock("@litelens/design-system", async (importOriginal) => {
  const actual = (await importOriginal()) as Record<string, unknown>;
  return {
    ...actual,
    renderErrorToast: ErrorToastSpy,
  };
});

// ─── Helpers ──────────────────────────────────────────────────────────────────
function fireForbiddenEvent(resource: string) {
  act(() => {
    getHandler()?.(resource);
  });
}

// ─── Tests ────────────────────────────────────────────────────────────────────
describe("useCatchForbiddenResources", () => {
  beforeEach(() => {
    setHandler(null);
    ErrorToastSpy.mockClear();
    isResourceForbiddenMock.mockClear();
  });

  // ── Scenario A ──────────────────────────────────────────────────────────────
  describe("Scenario A: closed drawer (no labelMap, resourceName=null, open=false)", () => {
    it("does NOT show a toast when a forbidden event arrives for the active resource", () => {
      renderHook(() =>
        useCatchForbiddenResources("nodes", {
          open: false,
          resourceName: null,
          onForbiddenDetected: vi.fn(),
        })
      );

      fireForbiddenEvent("nodes");

      expect(ErrorToastSpy).not.toHaveBeenCalled();
    });

    it("still records the resource in the forbiddenResources set", () => {
      const { result } = renderHook(() =>
        useCatchForbiddenResources("nodes", {
          open: false,
          resourceName: null,
        })
      );

      fireForbiddenEvent("nodes");

      expect(result.current.forbiddenResources.has("nodes")).toBe(true);
    });
  });

  // ── Scenario B ──────────────────────────────────────────────────────────────
  describe("Scenario B: open drawer (open=true, resourceName present)", () => {
    it("shows a specific 'cannot get' toast with the resource name", () => {
      renderHook(() =>
        useCatchForbiddenResources("nodes", {
          open: true,
          resourceName: "my-node",
          resourceLabel: "Node",
          onForbiddenDetected: vi.fn(),
        })
      );

      fireForbiddenEvent("nodes");

      expect(ErrorToastSpy).toHaveBeenCalledOnce();
      expect(ErrorToastSpy).toHaveBeenCalledWith({
        title: 'Access denied: cannot get Node "my-node"',
      });
    });

    it("calls onForbiddenDetected when the drawer's resource is forbidden", () => {
      const onForbiddenDetected = vi.fn();
      renderHook(() =>
        useCatchForbiddenResources("nodes", {
          open: true,
          resourceName: "my-node",
          resourceLabel: "Node",
          onForbiddenDetected,
        })
      );

      fireForbiddenEvent("nodes");

      expect(onForbiddenDetected).toHaveBeenCalledOnce();
    });

    it("falls back to activeResource as the label when resourceLabel is omitted", () => {
      renderHook(() =>
        useCatchForbiddenResources("nodes", {
          open: true,
          resourceName: "my-node",
          onForbiddenDetected: vi.fn(),
        })
      );

      fireForbiddenEvent("nodes");

      expect(ErrorToastSpy).toHaveBeenCalledWith({
        title: 'Access denied: cannot get nodes "my-node"',
      });
    });
  });

  // ── Scenario C ──────────────────────────────────────────────────────────────
  describe("Scenario C: list-view caller (labelMap present, no resourceName)", () => {
    const LABEL_MAP: Record<string, string> = {
      nodes: "Nodes",
      pods: "Pods",
    };

    it("shows a human-readable 'cannot list' toast using the labelMap", () => {
      renderHook(() => useCatchForbiddenResources("nodes", { labelMap: LABEL_MAP }));

      fireForbiddenEvent("nodes");

      expect(ErrorToastSpy).toHaveBeenCalledOnce();
      expect(ErrorToastSpy).toHaveBeenCalledWith({
        title: "Access denied: cannot list Nodes",
      });
    });

    it("does NOT toast for an event whose resource does not match the active resource", () => {
      // activeResource='nodes', but the event fires for 'secrets'
      // The early return `if (activeResourceRef.current !== resource)` blocks it.
      renderHook(() => useCatchForbiddenResources("nodes", { labelMap: LABEL_MAP }));

      fireForbiddenEvent("secrets");

      expect(ErrorToastSpy).not.toHaveBeenCalled();
    });

    it("does NOT call onForbiddenDetected (list-view branch has no callback path)", () => {
      const onForbiddenDetected = vi.fn();
      renderHook(() =>
        useCatchForbiddenResources("nodes", {
          labelMap: LABEL_MAP,
          onForbiddenDetected,
        })
      );

      fireForbiddenEvent("nodes");

      expect(onForbiddenDetected).not.toHaveBeenCalled();
    });
  });

  // ── Scenario D: different resource ──────────────────────────────────────────
  describe("Scenario D: event for a different resource", () => {
    it("ignores events that do not match the active resource (no toast, no callback)", () => {
      const onForbiddenDetected = vi.fn();
      const { result } = renderHook(() =>
        useCatchForbiddenResources("nodes", {
          open: true,
          resourceName: "my-node",
          resourceLabel: "Node",
          onForbiddenDetected,
        })
      );

      fireForbiddenEvent("pods"); // different resource

      expect(ErrorToastSpy).not.toHaveBeenCalled();
      expect(onForbiddenDetected).not.toHaveBeenCalled();
      // set still accumulates mismatched resources (setForbiddenResources fires before the guard)
      expect(result.current.forbiddenResources.has("pods")).toBe(true);
    });
  });

  // ── Scenario E: forbiddenResources set ──────────────────────────────────────
  describe("Scenario E: forbiddenResources set accumulates regardless of branch", () => {
    it("accumulates all forbidden resources from any caller pattern", () => {
      const { result } = renderHook(() =>
        useCatchForbiddenResources("nodes", {
          open: true,
          resourceName: "my-node",
          resourceLabel: "Node",
          onForbiddenDetected: vi.fn(),
        })
      );

      fireForbiddenEvent("pods");
      fireForbiddenEvent("nodes");

      expect(result.current.forbiddenResources.has("pods")).toBe(true);
      expect(result.current.forbiddenResources.has("nodes")).toBe(true);
    });
  });

  // ── Regression guard: the original bug ─────────────────────────────────────
  describe("Regression: closed drawer must NOT toast (original bug)", () => {
    it("no toast fires when opts has no labelMap and resourceName is null", () => {
      // Pre-fix: `else if (!opts?.resourceName)` matched this caller and fired
      // toast.custom. Post-fix: guard `opts?.labelMap &&` prevents the branch.
      const { result } = renderHook(() =>
        useCatchForbiddenResources("nodes", {
          open: false,
          resourceName: null,
        })
      );

      fireForbiddenEvent("nodes");

      expect(ErrorToastSpy).not.toHaveBeenCalled();
      expect(result.current.forbiddenResources.has("nodes")).toBe(true);
    });

    it("no toast fires when opts is entirely undefined", () => {
      renderHook(() => useCatchForbiddenResources("nodes"));

      fireForbiddenEvent("nodes");

      expect(ErrorToastSpy).not.toHaveBeenCalled();
    });
  });
});

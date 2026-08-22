import { describe, it, expect, beforeEach } from "vitest";
import { renderHook, act } from "@testing-library/react";
import type { NavEntry } from "@litelens/core";
import { usePluginNavEntries } from "../usePluginNavEntries";
import { clearNavRegistry, registerNavEntry, unregisterNavEntry } from "../pluginNavRegistry";

const groupEntry: NavEntry<string> = {
  kind: "group",
  group: {
    id: "helm",
    label: "Helm",
    icon: (() => null) as never,
    items: [
      { id: "helm-charts", label: "Charts", view: "helm-charts" },
      { id: "helm-releases", label: "Releases", view: "helm-releases" },
    ],
  },
};

const itemEntry: NavEntry<string> = {
  kind: "item",
  icon: (() => null) as never,
  item: { id: "kube-view", label: "Kube", view: "kube-view" },
};

describe("usePluginNavEntries", () => {
  beforeEach(() => {
    clearNavRegistry();
  });

  it("returns empty derived data when nothing is registered", () => {
    const { result } = renderHook(() => usePluginNavEntries());
    expect(result.current).toEqual({
      navEntries: [],
      viewTypeToPluginId: {},
      resourceLabels: {},
    });
  });

  it("derives viewType maps from a group entry", () => {
    act(() => registerNavEntry("helm", groupEntry));
    const { result } = renderHook(() => usePluginNavEntries());

    expect(result.current.navEntries).toEqual([groupEntry]);
    expect(result.current.viewTypeToPluginId).toEqual({
      "helm-charts": "helm",
      "helm-releases": "helm",
    });
    expect(result.current.resourceLabels).toEqual({
      "helm-charts": "Charts",
      "helm-releases": "Releases",
    });
  });

  it("derives viewType maps from an item entry", () => {
    act(() => registerNavEntry("kube", itemEntry));
    const { result } = renderHook(() => usePluginNavEntries());

    expect(result.current.viewTypeToPluginId).toEqual({ "kube-view": "kube" });
  });

  it("re-renders reactively when a plugin registers or unregisters", () => {
    const { result } = renderHook(() => usePluginNavEntries());
    expect(result.current.navEntries).toHaveLength(0);

    act(() => registerNavEntry("helm", groupEntry));
    expect(result.current.navEntries).toHaveLength(1);

    act(() => unregisterNavEntry("helm"));
    expect(result.current.navEntries).toHaveLength(0);
  });
});

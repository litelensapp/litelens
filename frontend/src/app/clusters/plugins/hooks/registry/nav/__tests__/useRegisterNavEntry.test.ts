import { describe, it, expect, beforeEach } from "vitest";
import { renderHook } from "@testing-library/react";
import type { NavEntry } from "@litelens/core";
import { useRegisterNavEntry } from "../useRegisterNavEntry";
import { clearNavRegistry, getNavEntries } from "../pluginNavRegistry";

const navEntry: NavEntry<string> = {
  kind: "group",
  group: {
    id: "helm",
    label: "Helm",
    icon: (() => null) as never,
    items: [{ id: "helm-charts", label: "Charts", view: "helm-charts" }],
  },
};

describe("useRegisterNavEntry", () => {
  beforeEach(() => {
    clearNavRegistry();
  });

  it("registers the nav entry on mount", () => {
    renderHook(() => useRegisterNavEntry("helm", "Helm", navEntry));
    expect(getNavEntries()).toEqual([{ pluginId: "helm", pluginName: "Helm", navEntry }]);
  });

  it("unregisters on unmount", () => {
    const { unmount } = renderHook(() => useRegisterNavEntry("helm", "Helm", navEntry));
    expect(getNavEntries()).toHaveLength(1);

    unmount();
    expect(getNavEntries()).toEqual([]);
  });

  it("does nothing when navEntry is undefined", () => {
    renderHook(() => useRegisterNavEntry("helm", "Helm", undefined));
    expect(getNavEntries()).toEqual([]);
  });
});

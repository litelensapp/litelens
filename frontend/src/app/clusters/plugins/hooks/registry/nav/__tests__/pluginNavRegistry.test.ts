import { describe, it, expect, beforeEach, vi } from "vitest";
import type { NavEntry } from "@litelens/core";
import {
  clearNavRegistry,
  getNavEntries,
  registerNavEntry,
  subscribeNavRegistry,
  unregisterNavEntry,
} from "../pluginNavRegistry";

const navEntry: NavEntry<string> = {
  kind: "item",
  icon: (() => null) as never,
  item: { id: "helm", label: "Helm", view: "helm-releases" },
};

describe("pluginNavRegistry", () => {
  beforeEach(() => {
    clearNavRegistry();
  });

  it("registers a nav entry keyed by pluginId", () => {
    registerNavEntry("helm", navEntry);
    expect(getNavEntries()).toEqual([{ pluginId: "helm", navEntry }]);
  });

  it("overwrites a previous registration for the same pluginId", () => {
    registerNavEntry("helm", navEntry);
    const updated: NavEntry<string> = { ...navEntry, item: { ...navEntry.item, label: "Helm v2" } };
    registerNavEntry("helm", updated);

    const entries = getNavEntries();
    expect(entries).toHaveLength(1);
    expect(entries[0].navEntry).toEqual(updated);
  });

  it("removes a registration on unregister", () => {
    registerNavEntry("helm", navEntry);
    unregisterNavEntry("helm");
    expect(getNavEntries()).toEqual([]);
  });

  it("notifies subscribers on register/unregister but not on a no-op unregister", () => {
    const listener = vi.fn();
    const unsubscribe = subscribeNavRegistry(listener);

    registerNavEntry("helm", navEntry);
    expect(listener).toHaveBeenCalledTimes(1);

    unregisterNavEntry("nonexistent-plugin");
    expect(listener).toHaveBeenCalledTimes(1);

    unregisterNavEntry("helm");
    expect(listener).toHaveBeenCalledTimes(2);

    unsubscribe();
    registerNavEntry("kube", navEntry);
    expect(listener).toHaveBeenCalledTimes(2);
  });

  it("clearNavRegistry empties the registry and notifies once", () => {
    const listener = vi.fn();
    subscribeNavRegistry(listener);

    registerNavEntry("helm", navEntry);
    registerNavEntry("kube", navEntry);
    listener.mockClear();

    clearNavRegistry();
    expect(getNavEntries()).toEqual([]);
    expect(listener).toHaveBeenCalledTimes(1);
  });
});

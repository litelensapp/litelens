import { describe, it, expect, beforeEach, vi } from "vitest";
import type { ComponentType } from "react";
import type { SharedUnifiedTrayContentProps } from "@litelens/core";
import {
  clearTrayRegistry,
  getTrayFamilies,
  registerTrayFamilies,
  subscribeTrayRegistry,
  unregisterTrayFamilies,
} from "../pluginTrayRegistry";

const TrayComponent = (() => null) as ComponentType<SharedUnifiedTrayContentProps>;
const families = { "helm-chart": TrayComponent };

describe("pluginTrayRegistry", () => {
  beforeEach(() => {
    clearTrayRegistry();
  });

  it("registers tray families keyed by pluginId", () => {
    registerTrayFamilies("helm", families);
    expect(getTrayFamilies()).toEqual([{ pluginId: "helm", families }]);
  });

  it("overwrites a previous registration for the same pluginId", () => {
    registerTrayFamilies("helm", families);
    const updated = { "helm-chart-upgrade": TrayComponent };
    registerTrayFamilies("helm", updated);

    const entries = getTrayFamilies();
    expect(entries).toHaveLength(1);
    expect(entries[0].families).toEqual(updated);
  });

  it("removes a registration on unregister", () => {
    registerTrayFamilies("helm", families);
    unregisterTrayFamilies("helm");
    expect(getTrayFamilies()).toEqual([]);
  });

  it("notifies subscribers on register/unregister but not on a no-op unregister", () => {
    const listener = vi.fn();
    const unsubscribe = subscribeTrayRegistry(listener);

    registerTrayFamilies("helm", families);
    expect(listener).toHaveBeenCalledTimes(1);

    unregisterTrayFamilies("nonexistent-plugin");
    expect(listener).toHaveBeenCalledTimes(1);

    unregisterTrayFamilies("helm");
    expect(listener).toHaveBeenCalledTimes(2);

    unsubscribe();
    registerTrayFamilies("kube", families);
    expect(listener).toHaveBeenCalledTimes(2);
  });

  it("clearTrayRegistry empties the registry and notifies once", () => {
    const listener = vi.fn();
    subscribeTrayRegistry(listener);

    registerTrayFamilies("helm", families);
    registerTrayFamilies("kube", families);
    listener.mockClear();

    clearTrayRegistry();
    expect(getTrayFamilies()).toEqual([]);
    expect(listener).toHaveBeenCalledTimes(1);
  });
});

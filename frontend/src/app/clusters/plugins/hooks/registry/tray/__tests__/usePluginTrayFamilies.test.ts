import { describe, it, expect, beforeEach } from "vitest";
import { renderHook, act } from "@testing-library/react";
import type { ComponentType } from "react";
import type { SharedUnifiedTrayContentProps } from "@litelens/core";
import { usePluginTrayFamilies } from "../usePluginTrayFamilies";
import {
  clearTrayRegistry,
  registerTrayFamilies,
  unregisterTrayFamilies,
} from "../pluginTrayRegistry";

const HelmTrayComponent = (() => null) as ComponentType<SharedUnifiedTrayContentProps>;
const KubeTrayComponent = (() => null) as ComponentType<SharedUnifiedTrayContentProps>;

describe("usePluginTrayFamilies", () => {
  beforeEach(() => {
    clearTrayRegistry();
  });

  it("returns an empty map when nothing is registered", () => {
    const { result } = renderHook(() => usePluginTrayFamilies());
    expect(result.current).toEqual({});
  });

  it("merges families from a single plugin", () => {
    act(() =>
      registerTrayFamilies("helm", {
        "helm-chart": HelmTrayComponent,
        "helm-chart-upgrade": HelmTrayComponent,
      })
    );
    const { result } = renderHook(() => usePluginTrayFamilies());

    expect(result.current).toEqual({
      "helm-chart": HelmTrayComponent,
      "helm-chart-upgrade": HelmTrayComponent,
    });
  });

  it("merges families across multiple plugins", () => {
    act(() => registerTrayFamilies("helm", { "helm-chart": HelmTrayComponent }));
    act(() => registerTrayFamilies("kube", { "kube-shell": KubeTrayComponent }));
    const { result } = renderHook(() => usePluginTrayFamilies());

    expect(result.current).toEqual({
      "helm-chart": HelmTrayComponent,
      "kube-shell": KubeTrayComponent,
    });
  });

  it("re-renders reactively when a plugin registers or unregisters", () => {
    const { result } = renderHook(() => usePluginTrayFamilies());
    expect(result.current).toEqual({});

    act(() => registerTrayFamilies("helm", { "helm-chart": HelmTrayComponent }));
    expect(result.current).toEqual({ "helm-chart": HelmTrayComponent });

    act(() => unregisterTrayFamilies("helm"));
    expect(result.current).toEqual({});
  });
});

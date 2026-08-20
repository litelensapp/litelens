import { describe, it, expect, beforeEach } from "vitest";
import { renderHook } from "@testing-library/react";
import type { ComponentType } from "react";
import type { SharedUnifiedTrayContentProps } from "@litelens/core";
import { useRegisterTrayFamilies } from "../useRegisterTrayFamilies";
import { clearTrayRegistry, getTrayFamilies } from "../pluginTrayRegistry";

const TrayComponent = (() => null) as ComponentType<SharedUnifiedTrayContentProps>;
const families = { "helm-chart": TrayComponent };

describe("useRegisterTrayFamilies", () => {
  beforeEach(() => {
    clearTrayRegistry();
  });

  it("registers the tray families on mount", () => {
    renderHook(() => useRegisterTrayFamilies("helm", families));
    expect(getTrayFamilies()).toEqual([{ pluginId: "helm", families }]);
  });

  it("unregisters on unmount", () => {
    const { unmount } = renderHook(() => useRegisterTrayFamilies("helm", families));
    expect(getTrayFamilies()).toHaveLength(1);

    unmount();
    expect(getTrayFamilies()).toEqual([]);
  });

  it("does nothing when families is undefined", () => {
    renderHook(() => useRegisterTrayFamilies("helm", undefined));
    expect(getTrayFamilies()).toEqual([]);
  });
});

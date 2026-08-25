import { render, waitFor } from "@testing-library/react";
import { beforeEach, describe, expect, it, vi } from "vitest";
import { PluginStylesheetsLoader } from "../PluginStylesheetsLoader";

const useGetInstalledPluginsMock = vi.hoisted(() => vi.fn());
const loadPluginModuleMock = vi.hoisted(() => vi.fn());
const getStylesheetsMock = vi.hoisted(() => vi.fn());
const getRegisteredPluginIdsMock = vi.hoisted(() => vi.fn());
const unregisterStylesheetsMock = vi.hoisted(() => vi.fn());
const ensurePluginStylesheetMock = vi.hoisted(() => vi.fn());

vi.mock("../../marketplace/hooks/data-access/useGetInstalledPlugins", () => ({
  useGetInstalledPlugins: useGetInstalledPluginsMock,
}));

vi.mock("../hooks/registry/stylesheet/pluginStylesheetRegistry", () => ({
  pluginStylesheetRegistry: {
    getStylesheets: getStylesheetsMock,
    getRegisteredPluginIds: getRegisteredPluginIdsMock,
    unregisterStylesheets: unregisterStylesheetsMock,
  },
}));

vi.mock("../utils/ensurePluginStylesheet", () => ({
  ensurePluginStylesheet: ensurePluginStylesheetMock,
}));

vi.mock("../utils/loadPluginModule", () => ({
  loadPluginModule: loadPluginModuleMock,
}));

describe("PluginStylesheetsLoader", () => {
  beforeEach(() => {
    vi.clearAllMocks();
    loadPluginModuleMock.mockResolvedValue({});
    ensurePluginStylesheetMock.mockResolvedValue(undefined);
    getRegisteredPluginIdsMock.mockReturnValue([]);
  });

  it("loads each ready plugin's bundle and injects its registered stylesheets", async () => {
    useGetInstalledPluginsMock.mockReturnValue({
      readyPlugins: [
        { pluginId: "helm", bundleChecksum: "abc123" },
        { pluginId: "kube", bundleChecksum: "def456" },
      ],
    });
    const helmStylesheets = [Promise.resolve({ default: ".helm {}" })];
    getStylesheetsMock.mockImplementation((pluginId: string) =>
      pluginId === "helm" ? helmStylesheets : []
    );

    render(<PluginStylesheetsLoader />);

    await waitFor(() => {
      expect(loadPluginModuleMock).toHaveBeenCalledWith("helm", "abc123");
      expect(loadPluginModuleMock).toHaveBeenCalledWith("kube", "def456");
    });
    await waitFor(() => {
      expect(ensurePluginStylesheetMock).toHaveBeenCalledWith("helm", helmStylesheets);
      expect(ensurePluginStylesheetMock).toHaveBeenCalledWith("kube", []);
    });
  });

  it("unregisters stylesheets for plugins no longer ready", () => {
    useGetInstalledPluginsMock.mockReturnValue({
      readyPlugins: [{ pluginId: "helm", bundleChecksum: "abc123" }],
    });
    getRegisteredPluginIdsMock.mockReturnValue(["helm", "kube"]);

    render(<PluginStylesheetsLoader />);

    expect(unregisterStylesheetsMock).toHaveBeenCalledWith("kube");
    expect(unregisterStylesheetsMock).not.toHaveBeenCalledWith("helm");
  });

  it("does nothing when there are no ready plugins", () => {
    useGetInstalledPluginsMock.mockReturnValue({ readyPlugins: [] });

    render(<PluginStylesheetsLoader />);

    expect(loadPluginModuleMock).not.toHaveBeenCalled();
    expect(ensurePluginStylesheetMock).not.toHaveBeenCalled();
  });

  it("logs and does not throw when loading a plugin bundle fails", async () => {
    const consoleErrorSpy = vi.spyOn(console, "error").mockImplementation(() => {});
    useGetInstalledPluginsMock.mockReturnValue({
      readyPlugins: [{ pluginId: "broken", bundleChecksum: "zzz" }],
    });
    loadPluginModuleMock.mockRejectedValue(new Error("network error"));

    render(<PluginStylesheetsLoader />);

    await waitFor(() => {
      expect(consoleErrorSpy).toHaveBeenCalledWith(
        expect.stringContaining("Failed to load stylesheets for plugin broken"),
        expect.any(Error)
      );
    });
    expect(ensurePluginStylesheetMock).not.toHaveBeenCalled();

    consoleErrorSpy.mockRestore();
  });
});

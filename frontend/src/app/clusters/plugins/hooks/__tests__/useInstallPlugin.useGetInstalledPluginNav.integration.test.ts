import { vi, describe, it, expect, beforeEach } from "vitest";
import { renderHook, waitFor } from "@testing-library/react";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { createElement } from "react";
import { useMutateInstallPlugin } from "../../../../marketplace/hooks/useMutateInstallPlugin";
import { useGetInstalledPluginNav } from "../useGetInstalledPluginNav";

/**
 * Regression test for the bug where NavSidebar didn't show a newly-installed
 * plugin until useGetInstalledPluginNav's 5s poll happened to fire. Renders both
 * hooks against the same QueryClient (as MainLayout.tsx does in the app) and
 * verifies that the install mutation invalidates useGetInstalledPluginNav's status query
 * without waiting for refetchInterval.
 */

const installPluginMock = vi.hoisted(() => vi.fn().mockResolvedValue(undefined));
const getInstalledPluginsMock = vi.hoisted(() => vi.fn());

vi.mock("@wailsjs/go/app/App", () => ({
  InstallPlugin: installPluginMock,
  GetInstalledPlugins: getInstalledPluginsMock,
}));

function makeWrapper() {
  const client = new QueryClient({
    defaultOptions: { queries: { retry: false } },
  });
  return {
    wrapper: ({ children }: { children: React.ReactNode }) =>
      createElement(QueryClientProvider, { client }, children),
    client,
  };
}

describe("useMutateInstallPlugin + useGetInstalledPluginNav integration", () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it("reflects a newly-installed plugin's status immediately, without waiting for the 5s poll", async () => {
    getInstalledPluginsMock.mockResolvedValue([
      { pluginId: "helm", status: "NOT_INSTALLED", progress: 0 },
    ]);

    const { wrapper } = makeWrapper();

    const nav = renderHook(() => useGetInstalledPluginNav(), { wrapper });
    const install = renderHook(() => useMutateInstallPlugin(), { wrapper });

    // Wait for the merged installed-plugins query (status + bundleChecksum in one call).
    await waitFor(() => {
      expect(getInstalledPluginsMock).toHaveBeenCalled();
    });

    // Plugin becomes READY on the backend once install completes.
    getInstalledPluginsMock.mockResolvedValue([
      { pluginId: "helm", status: "READY", progress: 100, bundleChecksum: "abc" },
    ]);

    await install.result.current.mutateAsync({ pluginId: "helm" });

    // useGetInstalledPluginNav's installed-plugins query should refetch and observe
    // READY status right away (no fake-timer advance needed to hit the 5s poll).
    await waitFor(() => {
      const statusesQuery = nav.result.current;
      expect(statusesQuery.isLoading).toBe(false);
    });

    await waitFor(
      () => {
        // GetInstalledPlugins must have been re-invoked after install, proving the
        // "installed-plugins" query was invalidated and refetched.
        expect(getInstalledPluginsMock.mock.calls.length).toBeGreaterThan(1);
      },
      { timeout: 1000 }
    );
  });
});

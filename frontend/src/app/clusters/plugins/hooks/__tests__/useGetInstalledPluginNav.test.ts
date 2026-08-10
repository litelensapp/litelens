import { vi, describe, it, expect, beforeEach } from "vitest";
import { QueryClient } from "@tanstack/react-query";

/**
 * Test suite for useGetInstalledPluginNav cache invalidation behavior.
 *
 * Key scenario: When useInstallPlugin invalidates the "installed-plugins" query key,
 * useGetInstalledPluginNav's query with queryKey ["installed-plugins"] should
 * be marked as stale and refetch on next access.
 */
describe("useGetInstalledPluginNav cache invalidation", () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it("should be invalidated when useInstallPlugin invalidates installed-plugins", async () => {
    const client = new QueryClient({
      defaultOptions: {
        queries: { retry: false },
      },
    });

    // Simulate the query used by useGetInstalledPluginNav
    const navQueryKey = ["installed-plugins"];

    // Pre-populate cache
    client.setQueryData(navQueryKey, [
      { pluginId: "helm-plugin", status: "NOT_INSTALLED" },
      { pluginId: "kube-plugin", status: "READY" },
    ]);

    // Verify cache is populated
    const cachedData = client.getQueryData(navQueryKey);
    expect(cachedData).toBeDefined();

    // Get the query object before invalidation
    const queryBefore = client.getQueryCache().find({ queryKey: navQueryKey });
    expect(queryBefore?.isStale()).toBe(false);

    // Now invalidate using the same prefix matching logic as useInstallPlugin
    // (queryKey: ["installed-plugins"])
    await client.invalidateQueries({
      queryKey: ["installed-plugins"],
    });

    // Verify the query is now stale
    const queryAfter = client.getQueryCache().find({ queryKey: navQueryKey });
    expect(queryAfter?.isStale()).toBe(true);
  });

  it("should not invalidate unrelated queries", async () => {
    const client = new QueryClient({
      defaultOptions: {
        queries: { retry: false },
      },
    });

    const pluginStatusQuery = ["plugin-status", "helm"];
    const navQuery = ["installed-plugins"];
    const otherQuery = ["plugin-nav-entries", "helm:abc123"];

    client.setQueryData(pluginStatusQuery, { status: "READY" });
    client.setQueryData(navQuery, [{ pluginId: "helm", status: "READY" }]);
    client.setQueryData(otherQuery, [{ view: "helm", label: "Helm" }]);

    // Invalidate only the installed-plugins query key (not the singular plugin-status key)
    await client.invalidateQueries({
      queryKey: ["installed-plugins"],
    });

    // installed-plugins should be stale
    expect(client.getQueryCache().find({ queryKey: navQuery })?.isStale()).toBe(true);

    // Other unrelated queries should NOT be affected
    expect(client.getQueryCache().find({ queryKey: pluginStatusQuery })?.isStale()).toBe(false);
    expect(client.getQueryCache().find({ queryKey: otherQuery })?.isStale()).toBe(false);
  });
});

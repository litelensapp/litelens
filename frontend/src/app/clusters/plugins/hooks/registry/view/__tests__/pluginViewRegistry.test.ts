import { describe, it, expect, beforeEach } from "vitest";
import type { ComponentType } from "react";
import {
  clearViewRegistry,
  getViewAssets,
  registerViews,
  unregisterView,
} from "../pluginViewRegistry";

const ViewComponent = (() => null) as ComponentType;
const config = {
  name: "helm-charts",
  component: ViewComponent,
  stylesheet: Promise.resolve({ default: ".custom-view { color: red; }" }),
};

describe("pluginViewRegistry", () => {
  beforeEach(() => {
    clearViewRegistry();
  });

  it("registers view assets keyed by pluginId", () => {
    registerViews("helm", [config]);
    expect(getViewAssets()).toEqual([
      {
        pluginId: "helm",
        name: "helm-charts",
        component: ViewComponent,
        stylesheet: config.stylesheet,
      },
    ]);
  });

  it("registers view assets without a stylesheet", () => {
    const configNoStylesheet = { name: "kube-view", component: ViewComponent };
    registerViews("kube", [configNoStylesheet]);
    expect(getViewAssets()).toEqual([
      { pluginId: "kube", name: "kube-view", component: ViewComponent },
    ]);
  });

  it("registers multiple view components for the same pluginId", () => {
    const second = {
      name: "helm-releases",
      component: ViewComponent,
      stylesheet: Promise.resolve({ default: ".second-view { color: green; }" }),
    };
    registerViews("helm", [config, second]);

    const assets = getViewAssets();
    expect(assets).toHaveLength(2);
    expect(assets).toEqual([
      {
        pluginId: "helm",
        name: "helm-charts",
        component: ViewComponent,
        stylesheet: config.stylesheet,
      },
      {
        pluginId: "helm",
        name: "helm-releases",
        component: ViewComponent,
        stylesheet: second.stylesheet,
      },
    ]);
  });

  it("overwrites a previous registration for the same pluginId", async () => {
    registerViews("helm", [config]);
    const updated = {
      name: "helm-charts",
      component: ViewComponent,
      stylesheet: Promise.resolve({ default: ".new-style { color: blue; }" }),
    };
    registerViews("helm", [updated]);

    const assets = getViewAssets();
    expect(assets).toHaveLength(1);
    await expect(assets[0].stylesheet).resolves.toEqual({ default: ".new-style { color: blue; }" });
  });

  it("removes a registration on unregister", () => {
    registerViews("helm", [config]);
    unregisterView("helm");
    expect(getViewAssets()).toEqual([]);
  });

  it("unregister on a nonexistent pluginId is a no-op", () => {
    registerViews("helm", [config]);
    unregisterView("nonexistent-plugin");
    expect(getViewAssets()).toEqual([
      {
        pluginId: "helm",
        name: "helm-charts",
        component: ViewComponent,
        stylesheet: config.stylesheet,
      },
    ]);
  });

  it("clearViewRegistry empties the registry", () => {
    registerViews("helm", [config]);
    registerViews("kube", [config]);

    clearViewRegistry();
    expect(getViewAssets()).toEqual([]);
  });

  it("returns multiple registered assets in the list", () => {
    registerViews("helm", [config]);
    registerViews("kube", [{ name: "kube-view", component: ViewComponent }]);
    registerViews("prometheus", [config]);

    const assets = getViewAssets();
    expect(assets).toHaveLength(3);
    expect(assets.map((a) => a.pluginId).sort()).toEqual(["helm", "kube", "prometheus"]);
  });
});

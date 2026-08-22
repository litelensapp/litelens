import { describe, it, expect, beforeEach } from "vitest";
import {
  clearStylesheetRegistry,
  getStylesheets,
  registerStylesheets,
  unregisterStylesheets,
} from "../pluginStylesheetRegistry";

describe("pluginStylesheetRegistry", () => {
  beforeEach(() => {
    clearStylesheetRegistry();
  });

  it("registers stylesheets keyed by pluginId", () => {
    const stylesheet = Promise.resolve({ default: ".helm { color: red; }" });
    registerStylesheets("helm", [stylesheet]);
    expect(getStylesheets("helm")).toEqual([stylesheet]);
  });

  it("returns an empty array for an unregistered pluginId", () => {
    expect(getStylesheets("nonexistent-plugin")).toEqual([]);
  });

  it("overwrites a previous registration for the same pluginId", () => {
    registerStylesheets("helm", [Promise.resolve({ default: ".old { color: red; }" })]);
    const updated = Promise.resolve({ default: ".new { color: blue; }" });
    registerStylesheets("helm", [updated]);

    expect(getStylesheets("helm")).toEqual([updated]);
  });

  it("removes a registration on unregister", () => {
    registerStylesheets("helm", [Promise.resolve({ default: ".helm { color: red; }" })]);
    unregisterStylesheets("helm");
    expect(getStylesheets("helm")).toEqual([]);
  });

  it("unregister on a nonexistent pluginId is a no-op", () => {
    const stylesheet = Promise.resolve({ default: ".helm { color: red; }" });
    registerStylesheets("helm", [stylesheet]);
    unregisterStylesheets("nonexistent-plugin");
    expect(getStylesheets("helm")).toEqual([stylesheet]);
  });

  it("clearStylesheetRegistry empties the registry", () => {
    registerStylesheets("helm", [Promise.resolve({ default: ".helm { color: red; }" })]);
    registerStylesheets("kube", [Promise.resolve({ default: ".kube { color: blue; }" })]);

    clearStylesheetRegistry();
    expect(getStylesheets("helm")).toEqual([]);
    expect(getStylesheets("kube")).toEqual([]);
  });

  it("keeps multiple stylesheets in registration order for one pluginId", () => {
    const first = Promise.resolve({ default: ".a {}" });
    const second = Promise.resolve({ default: ".b {}" });
    registerStylesheets("helm", [first, second]);
    expect(getStylesheets("helm")).toEqual([first, second]);
  });
});

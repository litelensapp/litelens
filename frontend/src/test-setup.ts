import "@testing-library/jest-dom/vitest";
import * as DesignSystem from "@litelens/design-system";
import * as React from "react";
import * as ReactJsxRuntime from "react/jsx-runtime";
import * as ReactDom from "react-dom";
import * as ReactQuery from "@tanstack/react-query";

// jsdom does not implement ResizeObserver — provide a no-op stub so any component
// that uses it (TruncatedText, usePodLogs, usePodExec) doesn't throw in tests.
globalThis.ResizeObserver = class ResizeObserver {
  observe() {}
  unobserve() {}
  disconnect() {}
};

// Mirrors the window.__LITELENS_VENDOR__ global that src/main.tsx sets up at
// app boot for plugin bundles' import-map-resolved bare specifiers (see
// frontend/public/vendor/*.js). Node/jsdom has no browser import-map support,
// so any test that dynamically imports a plugin bundle needs this global
// populated, or "react"/"@litelens/design-system" imports inside that bundle
// will throw.
// eslint-disable-next-line @typescript-eslint/no-explicit-any
(globalThis.window as any).__LITELENS_VENDOR__ = {
  react: React,
  reactDom: ReactDom,
  reactJsxRuntime: ReactJsxRuntime,
  designSystem: DesignSystem,
  reactQuery: ReactQuery,
};

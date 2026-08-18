import * as Core from "@litelens/core";
import * as DesignSystem from "@litelens/design-system";
import { TooltipProvider } from "@litelens/design-system";
import * as ReactQuery from "@tanstack/react-query";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import * as React from "react";
import * as ReactDom from "react-dom";
import { createRoot } from "react-dom/client";
import * as ReactJsxRuntime from "react/jsx-runtime";
import { App } from "./app/App";
import { useResourceLinks } from "./app/clusters/shared/hooks/useResourceLinks";
import "./style.css";

// Expose the host's own singleton module instances so dynamically-imported
// plugin bundles (built with react/react-dom/@litelens/design-system marked
// "external") can resolve those bare specifiers via the import map in
// index.html against the SAME instances the host uses, instead of bundling
// their own copies. A second React instance would break hooks/context when
// a plugin component is mounted inline in the host's fiber tree. Must run
// before any plugin can be dynamically imported — satisfied here since
// plugins only load on user navigation, well after this module executes.
declare global {
  interface Window {
    __LITELENS_VENDOR__: {
      react: typeof React;
      reactDom: typeof ReactDom;
      reactJsxRuntime: typeof ReactJsxRuntime;
      designSystem: typeof DesignSystem;
      core: typeof Core;
      reactQuery: typeof ReactQuery;
    };
  }
}

window.__LITELENS_VENDOR__ = {
  react: React,
  reactDom: ReactDom,
  reactJsxRuntime: ReactJsxRuntime,
  designSystem: DesignSystem,
  core: {
    ...Core,
    useResourceLinks,
  },
  reactQuery: ReactQuery,
};

const queryClient = new QueryClient();

const container = document.getElementById("root");

const root = createRoot(container!);

root.render(
  <React.StrictMode>
    <QueryClientProvider client={queryClient}>
      <TooltipProvider>
        <App />
      </TooltipProvider>
    </QueryClientProvider>
  </React.StrictMode>
);

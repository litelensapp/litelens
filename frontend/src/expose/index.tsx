import * as DesignSystem from "@litelens/design-system";
import * as ReactQuery from "@tanstack/react-query";
import * as React from "react";
import * as ReactDom from "react-dom";
import * as ReactJsxRuntime from "react/jsx-runtime";
import { registerEvents } from "../app/clusters/plugins/hooks/registry/event/pluginEventRegistry";
import { registerNavEntry } from "../app/clusters/plugins/hooks/registry/nav/pluginNavRegistry";
import { registerTrayFamilies } from "../app/clusters/plugins/hooks/registry/tray/pluginTrayRegistry";
import { registerViews } from "../app/clusters/plugins/hooks/registry/view/pluginViewRegistry";
import { registerStylesheets } from "../app/plugins/hooks/registry/stylesheet/pluginStylesheetRegistry";
import { queryClient } from "../queryClient";
import { useExposeMethods } from "./hooks/useExposeMethods";
import { useExposeProperties } from "./hooks/useExposeProperties";

// Exposed via appWideAPI.getQueryClient() so plugin code that isn't a
// mounted React component (e.g. index.ts's module-scope registration calls)
// can still build event handlers that invalidate queries, without needing
// useQueryClient()'s React context.
function getQueryClient(): typeof queryClient {
  return queryClient;
}

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
      core: {
        appWideAPI: {
          registerStylesheets: typeof registerStylesheets;
          getQueryClient: typeof getQueryClient;
        };
        clusterWideAPI: {
          useExposeProperties: typeof useExposeProperties;
          useExposeMethods: typeof useExposeMethods;
          registerViews: typeof registerViews;
          registerNavEntry: typeof registerNavEntry;
          registerTrayFamilies: typeof registerTrayFamilies;
          registerEvents: typeof registerEvents;
        };
      };
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
    appWideAPI: {
      registerStylesheets,
      getQueryClient,
    },
    clusterWideAPI: {
      useExposeProperties,
      useExposeMethods,
      registerViews,
      registerNavEntry,
      registerTrayFamilies,
      registerEvents,
    },
  },
  reactQuery: ReactQuery,
};

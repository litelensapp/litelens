import { TooltipProvider } from "@litelens/design-system";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import * as React from "react";
import { createRoot } from "react-dom/client";
import { App } from "./app/App";
import "./expose";
import "./style.css";

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

import { cn, LoadingSpinner } from "@litelens/design-system";
import { FC, lazy, Suspense, useCallback, useReducer } from "react";
import { usePodLogs } from "../hooks/usePodLogs";
import { PodLogTrayBottomBar } from "./PodLogTrayBottomBar";
import { PodLogTrayToolbar } from "./PodLogTrayToolbar";
import type { TrayTab } from "./PodTray";

const LogsPanel = lazy(() => import("./LogsPanel").then((m) => ({ default: m.LogsPanel })));

export interface LogTabOptions {
  container: string;
  wrap: boolean;
  showTimestamps: boolean;
  showPrevTerminated: boolean;
}

type LogState = {
  opts: LogTabOptions;
};

type LogAction = { type: "update_opts"; opts: LogTabOptions };

function logReducer(_: LogState, action: LogAction): LogState {
  return { opts: action.opts };
}

const initialLogState = (tab: TrayTab): LogState => ({
  opts: {
    container: tab.containers[0]?.Name ?? "",
    wrap: false,
    showTimestamps: false,
    showPrevTerminated: false,
  },
});

interface PodLogTrayContentProps {
  tab: TrayTab;
  collapsed: boolean;
}

export const PodLogTrayContent: FC<PodLogTrayContentProps> = ({ tab, collapsed }) => {
  const [state, dispatch] = useReducer(logReducer, tab, initialLogState);

  const {
    setContainerRef,
    status,
    error,
    clear,
    searchTerm,
    handleSearch,
    handleSearchNext,
    matchCount,
    currentMatchIdx,
  } = usePodLogs({
    contextName: tab.contextName,
    ns: tab.ns,
    pod: tab.pod,
    container: state.opts.container,
    wrap: state.opts.wrap,
  });

  const updateOpts = useCallback(
    <K extends keyof LogTabOptions>(key: K, value: LogTabOptions[K]) => {
      dispatch({ type: "update_opts", opts: { ...state.opts, [key]: value } });
    },
    [state.opts]
  );

  const handleContainerChange = useCallback(
    (c: string) => dispatch({ type: "update_opts", opts: { ...state.opts, container: c } }),
    [state.opts]
  );

  return (
    <>
      {/* [B] Toolbar row */}
      <PodLogTrayToolbar
        collapsed={collapsed}
        tab={tab}
        container={state.opts.container}
        onContainerChange={handleContainerChange}
        searchTerm={searchTerm}
        matchCount={matchCount}
        currentMatchIdx={currentMatchIdx}
        status={status}
        onClear={clear}
        onSearch={handleSearch}
        onSearchNext={handleSearchNext}
      />

      {/* [C] Content area */}
      <div className={cn("min-h-0 flex-1 overflow-hidden pl-4", collapsed && "hidden")}>
        <Suspense fallback={<LoadingSpinner />}>
          <LogsPanel
            containerRef={setContainerRef}
            status={status}
            error={error}
            wrap={state.opts.wrap}
          />
        </Suspense>
      </div>

      {/* [D] Bottom bar: logs options */}
      <PodLogTrayBottomBar collapsed={collapsed} opts={state.opts} updateOpts={updateOpts} />
    </>
  );
};

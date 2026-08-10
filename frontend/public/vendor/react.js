/*
 * Vendor shim: redirects the bare "react" specifier (left in place by plugin
 * bundlers that mark react as external) to the host app's own React
 * singleton, exposed on window.__LITELENS_VENDOR__ by src/main.tsx before
 * any plugin can be dynamically imported. This keeps hooks/context/portals
 * working correctly when a plugin component is mounted inline in the host's
 * fiber tree, since React requires exactly one module instance to do so.
 */
if (!window.__LITELENS_VENDOR__) {
  throw new Error(
    "window.__LITELENS_VENDOR__ is not set — src/main.tsx must run before any plugin is dynamically imported."
  );
}
const React = window.__LITELENS_VENDOR__.react;

export default React;

export const {
  Children,
  Component,
  Fragment,
  Profiler,
  PureComponent,
  StrictMode,
  Suspense,
  cloneElement,
  createContext,
  createElement,
  createFactory,
  createRef,
  forwardRef,
  isValidElement,
  lazy,
  memo,
  startTransition,
  unstable_act,
  use,
  useActionState,
  useCallback,
  useContext,
  useDebugValue,
  useDeferredValue,
  useEffect,
  useId,
  useImperativeHandle,
  useInsertionEffect,
  useLayoutEffect,
  useMemo,
  useOptimistic,
  useReducer,
  useRef,
  useState,
  useSyncExternalStore,
  useTransition,
  version,
} = React;

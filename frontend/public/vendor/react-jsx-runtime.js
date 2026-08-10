/*
 * Vendor shim for the "react/jsx-runtime" bare specifier. See react.js in
 * this directory for why this indirection exists.
 */
if (!window.__LITELENS_VENDOR__) {
  throw new Error(
    "window.__LITELENS_VENDOR__ is not set — src/main.tsx must run before any plugin is dynamically imported."
  );
}
const jsxRuntime = window.__LITELENS_VENDOR__.reactJsxRuntime;

export const jsx = jsxRuntime.jsx;
export const jsxs = jsxRuntime.jsxs;
export const Fragment = jsxRuntime.Fragment;

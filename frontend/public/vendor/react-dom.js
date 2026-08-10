/*
 * Vendor shim for the "react-dom" bare specifier. See react.js in this
 * directory for why this indirection exists.
 */
if (!window.__LITELENS_VENDOR__) {
  throw new Error(
    "window.__LITELENS_VENDOR__ is not set — src/main.tsx must run before any plugin is dynamically imported."
  );
}
const ReactDOM = window.__LITELENS_VENDOR__.reactDom;

export default ReactDOM;

export const {
  createPortal,
  findDOMNode,
  flushSync,
  hydrate,
  render,
  unmountComponentAtNode,
  unstable_batchedUpdates,
  version,
} = ReactDOM;

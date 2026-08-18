/*
 * Vendor shim for the "@litelens/core" bare specifier. See
 * ../react.js for why this indirection exists.
 *
 * This export list is generated from the host's actual @litelens/core
 * build (Object.keys() of an import of the package). Regenerate it whenever
 * the core package's public API surface changes.
 */
if (!window.__LITELENS_VENDOR__) {
  throw new Error(
    "window.__LITELENS_VENDOR__ is not set — src/main.tsx must run before any plugin is dynamically imported."
  );
}
const Core = window.__LITELENS_VENDOR__.core;

export const {
  useResourceLinks,
} = Core;

// Shared boundary type between the main app and plugin frontends (e.g.
// plugins/helm). Plugin bundles build standalone and cannot import from the
// main app's `@/app` module tree, so any type crossing that boundary lives
// here instead. Keep this to the minimal shape actually consumed across the
// boundary — the main app's `Namespace` is a structural superset and
// extends this type directly.

export interface SharedNamespaceContext {
  Name: string;
}

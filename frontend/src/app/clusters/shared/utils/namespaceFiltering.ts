/**
 * Shared utilities for multi-namespace filtering logic.
 * Used across data-access hooks and event handlers.
 */

/**
 * Derives the effective namespace to pass to backend hooks.
 * - If namespaces.length === 1: return that namespace (preserve single-namespace path)
 * - Otherwise (0 or 2+): return "" (cluster-wide)
 */
export function getEffectiveNamespace(namespaces: string[]): string {
  if (namespaces.length === 1) {
    return namespaces[0];
  }
  return "";
}

/**
 * Filters items by namespace membership.
 * - If namespaces.length === 0: return all items (no filter)
 * - Otherwise: return items whose Namespace is in the set
 */
export function filterByNamespaces<T extends { Namespace: string }>(
  items: T[],
  namespaces: string[]
): T[] {
  if (namespaces.length === 0) {
    return items;
  }
  const namespaceSet = new Set(namespaces);
  return items.filter((item) => namespaceSet.has(item.Namespace));
}

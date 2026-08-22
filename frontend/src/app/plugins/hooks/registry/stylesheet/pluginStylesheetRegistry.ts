const registry = new Map<string, Array<Promise<{ default: string }>>>();

export function registerStylesheets(
  pluginId: string,
  stylesheets: Array<Promise<{ default: string }>>
): void {
  registry.set(pluginId, stylesheets);
}

export function unregisterStylesheets(pluginId: string): void {
  registry.delete(pluginId);
}

export function getStylesheets(pluginId: string): Array<Promise<{ default: string }>> {
  return registry.get(pluginId) ?? [];
}

export function clearStylesheetRegistry(): void {
  registry.clear();
}

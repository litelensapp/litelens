import type { ComponentType } from "react";

export interface ViewAssets {
  pluginId: string;
  name: string;
  component: ComponentType;
  stylesheet?: Promise<{ default: string }>;
}

const registry = new Map<string, ViewAssets[]>();

export function registerViews(
  pluginId: string,
  configs: Array<{
    name: string;
    component: ComponentType;
    stylesheet?: Promise<{ default: string }>;
  }>
): void {
  registry.set(
    pluginId,
    configs.map((config) => ({ pluginId, ...config }))
  );
}

export function unregisterView(pluginId: string): void {
  registry.delete(pluginId);
}

export function getViewAssets(): ViewAssets[] {
  return Array.from(registry.values()).flat();
}

export function clearViewRegistry(): void {
  registry.clear();
}

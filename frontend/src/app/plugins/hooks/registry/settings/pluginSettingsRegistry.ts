import type { PluginSettingsTab } from "@litelens/core";
import { useSyncExternalStore } from "react";

type Listener = () => void;

class PluginSettingsRegistry {
  private readonly registry = new Map<string, PluginSettingsTab>();
  private readonly listeners = new Set<Listener>();

  private snapshot: PluginSettingsTab[] = [];

  registerSettingsTab(pluginId: string, tab: PluginSettingsTab): void {
    this.registry.set(pluginId, tab);
    this.notify();
  }

  unregisterSettingsTab(pluginId: string): void {
    if (this.registry.delete(pluginId)) {
      this.notify();
    }
  }

  getSettingsTabs(): PluginSettingsTab[] {
    return this.snapshot;
  }

  subscribeSettingsRegistry(listener: Listener): () => void {
    this.listeners.add(listener);
    return () => this.listeners.delete(listener);
  }

  clearRegistry(): void {
    if (this.registry.size > 0) {
      this.registry.clear();
      this.notify();
    }
  }

  private notify(): void {
    this.snapshot = Array.from(this.registry.values());
    for (const listener of this.listeners) {
      listener();
    }
  }
}

export const pluginSettingsRegistry = new PluginSettingsRegistry();

export function usePluginSettingsTabs(): PluginSettingsTab[] {
  return useSyncExternalStore(
    pluginSettingsRegistry.subscribeSettingsRegistry.bind(pluginSettingsRegistry),
    pluginSettingsRegistry.getSettingsTabs.bind(pluginSettingsRegistry)
  );
}

import type { ComponentType } from "react";
import type { SharedUnifiedTrayContentProps } from "@litelens/core";

type Listener = () => void;

export interface RegisteredTrayFamilies {
  pluginId: string;
  families: Record<string, ComponentType<SharedUnifiedTrayContentProps>>;
}

const registry = new Map<string, RegisteredTrayFamilies>();
const listeners = new Set<Listener>();

// useSyncExternalStore requires getSnapshot to return a referentially stable
// value between mutations (React re-invokes it on every render to detect
// changes via Object.is) — a fresh Array.from() on every call fails that
// check and causes an infinite render loop. Cache the array here and only
// rebuild it when the registry actually changes.
let snapshot: RegisteredTrayFamilies[] = [];

export function registerTrayFamilies(
  pluginId: string,
  families: Record<string, ComponentType<SharedUnifiedTrayContentProps>>
): void {
  registry.set(pluginId, { pluginId, families });
  notify();
}

export function unregisterTrayFamilies(pluginId: string): void {
  if (registry.delete(pluginId)) {
    notify();
  }
}

export function getTrayFamilies(): RegisteredTrayFamilies[] {
  return snapshot;
}

export function subscribeTrayRegistry(listener: Listener): () => void {
  listeners.add(listener);
  return () => listeners.delete(listener);
}

export function clearTrayRegistry(): void {
  if (registry.size > 0) {
    registry.clear();
    notify();
  }
}

function notify(): void {
  snapshot = Array.from(registry.values());
  for (const listener of listeners) {
    listener();
  }
}

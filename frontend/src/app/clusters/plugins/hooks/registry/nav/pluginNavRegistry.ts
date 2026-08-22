import type { NavEntry } from "@litelens/core";

type Listener = () => void;

export interface RegisteredNavEntry {
  pluginId: string;
  navEntry: NavEntry<string>;
}

const registry = new Map<string, RegisteredNavEntry>();
const listeners = new Set<Listener>();

// useSyncExternalStore requires getSnapshot to return a referentially stable
// value between mutations (React re-invokes it on every render to detect
// changes via Object.is) — a fresh Array.from() on every call fails that
// check and causes an infinite render loop. Cache the array here and only
// rebuild it when the registry actually changes.
let snapshot: RegisteredNavEntry[] = [];

export function registerNavEntry(pluginId: string, navEntry: NavEntry<string>): void {
  registry.set(pluginId, { pluginId, navEntry });
  notify();
}

export function unregisterNavEntry(pluginId: string): void {
  if (registry.delete(pluginId)) {
    notify();
  }
}

export function getNavEntries(): RegisteredNavEntry[] {
  return snapshot;
}

export function subscribeNavRegistry(listener: Listener): () => void {
  listeners.add(listener);
  return () => listeners.delete(listener);
}

export function clearNavRegistry(): void {
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

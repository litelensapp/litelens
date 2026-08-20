const registry = new Map<string, (payload: unknown) => void>();

export function registerHandler(eventName: string, handler: (payload: unknown) => void): void {
  registry.set(eventName, handler);
}

export function getHandler(eventName: string): ((payload: unknown) => void) | undefined {
  return registry.get(eventName);
}

export function clearRegistry(): void {
  registry.clear();
}

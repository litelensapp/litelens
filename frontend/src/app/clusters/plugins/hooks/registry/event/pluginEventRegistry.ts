interface RegisteredEventHandler {
  pluginId: string;
  handler: (payload: unknown) => void;
}

const registry = new Map<string, RegisteredEventHandler>();

export function registerEvents(
  pluginId: string,
  // Plugins type each handler's payload with its own event-specific shape
  // rather than `unknown`, so this must accept `any` here — a `(payload:
  // unknown) => void` signature would reject those narrower handlers.
  // eslint-disable-next-line @typescript-eslint/no-explicit-any
  handlers: Record<string, (payload: any) => void>
): void {
  for (const [eventName, handler] of Object.entries(handlers)) {
    registry.set(eventName, { pluginId, handler });
  }
}

export function unregisterEvents(pluginId: string): void {
  for (const [eventName, entry] of registry) {
    if (entry.pluginId === pluginId) {
      registry.delete(eventName);
    }
  }
}

export function getHandler(eventName: string): ((payload: unknown) => void) | undefined {
  return registry.get(eventName)?.handler;
}

import { useMemo, useSyncExternalStore } from "react";
import type { UnifiedTrayContentComponent } from "../../../../shared/components/trays/unified/UnifiedTrayTypes";
import { getTrayFamilies, subscribeTrayRegistry } from "./pluginTrayRegistry";

/**
 * Reactive read side of pluginTrayRegistry — merges every registered plugin's
 * tray-family content components into a single family→component map,
 * mirroring usePluginNavEntries. Rebuilds only when a plugin registers or
 * unregisters via registerTrayFamilies/unregisterTrayFamilies.
 */
export function usePluginTrayFamilies(): Record<string, UnifiedTrayContentComponent> {
  const registered = useSyncExternalStore(subscribeTrayRegistry, getTrayFamilies, getTrayFamilies);

  return useMemo(
    () =>
      Object.assign({}, ...registered.map((r) => r.families)) as Record<
        string,
        UnifiedTrayContentComponent
      >,
    [registered]
  );
}

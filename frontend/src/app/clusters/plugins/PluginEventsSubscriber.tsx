import { FC } from "react";
import { useClusterWideEventListener } from "../shared/hooks/registry/event/useClusterWideEventListener";
import { usePluginDisabledEventSubscription } from "./hooks/usePluginDisabledEventSubscription";

// Renders inside MainLayoutProvider (which mounts UnifiedTrayProvider) so
// usePluginDisabledEventSubscription's useUnifiedTray() has a provider in
// its ancestor tree — this component itself doesn't render anything.
export const PluginEventsSubscriber: FC = () => {
  useClusterWideEventListener();
  usePluginDisabledEventSubscription();
  return null;
};

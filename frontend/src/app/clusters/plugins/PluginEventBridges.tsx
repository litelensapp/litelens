import { Component, FC, ReactNode, Suspense, lazy, useMemo } from "react";
import { useGetInstalledPlugins } from "../../marketplace/hooks/useGetInstalledPlugins";

class SilentErrorBoundary extends Component<{ children: ReactNode }, { hasError: boolean }> {
  constructor(props: { children: ReactNode }) {
    super(props);
    this.state = { hasError: false };
  }

  static getDerivedStateFromError() {
    return { hasError: true };
  }

  componentDidCatch(error: Error) {
    console.error("Plugin event bridge crashed:", error);
  }

  render() {
    return this.state.hasError ? null : this.props.children;
  }
}

const PluginEventBridgeLoader: FC<{ pluginId: string; bundleChecksum?: string }> = ({
  pluginId,
  bundleChecksum,
}) => {
  const cacheVersion = bundleChecksum?.substring(0, 8) || "unknown";
  const bundleUrl = `/api/plugins/${pluginId}/dist/index.js?v=${cacheVersion}`;

  const EventBridge = useMemo(
    () =>
      lazy(() =>
        // eslint-disable-next-line @typescript-eslint/no-explicit-any
        import(/* @vite-ignore */ bundleUrl as any).then((m: any) => ({
          default: m.PluginEventBridge ?? (() => null),
        }))
      ),
    [bundleUrl]
  );

  return (
    <SilentErrorBoundary>
      <Suspense fallback={null}>
        {/* EventBridge is memoized on bundleUrl (see above), not recreated on every render */}
        {/* eslint-disable-next-line react-hooks/static-components */}
        <EventBridge />
      </Suspense>
    </SilentErrorBoundary>
  );
};

/**
 * Mounts each READY plugin's optional event-listener bridge (e.g. toasts and
 * cache invalidation for backend operations the plugin itself triggers), so
 * the host never needs static, build-time knowledge of any plugin's hooks.
 * Mounted once per plugin regardless of which view is currently active, so
 * events fired while the user has navigated away from the plugin's own view
 * are still caught.
 */
export const PluginEventBridges: FC = () => {
  const { readyPlugins } = useGetInstalledPlugins();
  return (
    <>
      {readyPlugins.map((status) => {
        const { pluginId, bundleChecksum } = status;
        return (
          <PluginEventBridgeLoader
            key={pluginId}
            pluginId={pluginId}
            bundleChecksum={bundleChecksum}
          />
        );
      })}
    </>
  );
};

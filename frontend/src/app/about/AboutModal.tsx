import logo from "@/assets/images/logo-universal.png";
import { Button, Dialog, DialogContent, Divider, Loader2Icon } from "@litelens/design-system";
import { BrowserOpenURL } from "@wailsjs/runtime/runtime";
import { FC } from "react";
import pkg from "../../../package.json";
import { useGetInstalledPlugins } from "../marketplace/hooks/useGetInstalledPlugins";
import { formatBytes } from "../marketplace/utils/formatBytes";
import { useCheckForUpdate } from "./hooks/useCheckForUpdate";

// Strip leading non-numeric characters (^, ~, v, "go", etc.)
function clean(v: string): string {
  return v.replace(/^[^\d]+/, "");
}

const TECH: Array<{ label: string; version: string }> = [
  { label: "React", version: clean(pkg.dependencies.react) },
  { label: "Vite", version: clean(pkg.devDependencies.vite) },
  { label: "Tailwind CSS", version: clean(pkg.devDependencies.tailwindcss) },
];

interface Payload {
  version: string;
  go: string;
  wails: string;
  appSizeBytes: string;
}

interface Props {
  payload: Payload;
  onClose: () => void;
}

export const AboutModal: FC<Props> = ({ payload, onClose }) => {
  const { checkingForUpdate, checkForUpdate } = useCheckForUpdate();

  const runtimeTech = [
    { label: "Go", version: clean(payload.go) },
    { label: "Wails", version: clean(payload.wails) },
    { label: "Node", version: clean(__NODE_VERSION__) },
    ...TECH,
  ];

  const { pluginStatuses } = useGetInstalledPlugins();
  const installedPlugins = pluginStatuses.filter((p) => p.size !== undefined);

  const appSizeBytes = Number(payload.appSizeBytes);
  const hasAppSize =
    payload.appSizeBytes !== "" && !Number.isNaN(appSizeBytes) && appSizeBytes >= 0;
  const totalBytes =
    (hasAppSize ? appSizeBytes : 0) + installedPlugins.reduce((sum, p) => sum + (p.size ?? 0), 0);
  const hasSizeInfo = hasAppSize || installedPlugins.length > 0;

  return (
    <Dialog
      open
      onOpenChange={(open) => {
        if (!open) onClose();
      }}
    >
      <DialogContent aria-label="About LiteLens" size="sm">
        <div className="flex flex-col items-center px-8 pb-6 pt-8 text-center">
          <img src={logo} alt="LiteLens" className="mb-4 h-16 w-16 rounded-2xl" />

          <h1 className="text-lg font-semibold">LiteLens</h1>
          <p className="text-muted-foreground mt-0.5 font-mono text-xs">{payload.version}</p>

          <Divider className="my-5" />

          {/* Built with */}
          <div className="w-full text-left">
            <p className="text-muted-foreground mb-2 text-xs font-medium uppercase tracking-wider">
              Built with
            </p>
            <div className="flex flex-col gap-1">
              {runtimeTech.map(({ label, version }) => (
                <div key={label} className="flex items-center justify-between">
                  <span className="text-muted-foreground text-xs">{label}</span>
                  <span className="font-mono text-xs">{version}</span>
                </div>
              ))}
            </div>
          </div>

          <Divider className="my-5" />

          {/* Author */}
          <div className="w-full text-left">
            <p className="text-muted-foreground mb-1 text-xs font-medium uppercase tracking-wider">
              Author
            </p>
            <p className="text-sm font-medium">{import.meta.env.VITE_AUTHOR_NAME || "-"}</p>
            {import.meta.env.VITE_AUTHOR_URL ? (
              <Button
                variant="link"
                onClick={() => BrowserOpenURL(import.meta.env.VITE_AUTHOR_URL)}
                className="text-muted-foreground hover:text-foreground mt-0.5 h-auto p-0 font-mono text-xs underline-offset-2"
              >
                {import.meta.env.VITE_AUTHOR_URL.replace(/^https?:\/\//, "")}
              </Button>
            ) : (
              "-"
            )}
          </div>

          {hasSizeInfo && (
            <>
              <Divider className="my-5" />

              {/* Storage */}
              <div className="w-full text-left">
                <p className="text-muted-foreground mb-2 text-xs font-medium uppercase tracking-wider">
                  Storage
                </p>
                <div className="flex flex-col gap-1">
                  {hasAppSize && (
                    <div className="flex items-center justify-between">
                      <span className="text-muted-foreground text-xs">App</span>
                      <span className="font-mono text-xs">{formatBytes(appSizeBytes)}</span>
                    </div>
                  )}
                  {installedPlugins.map((p) => (
                    <div key={p.pluginId} className="flex items-center justify-between">
                      <span className="text-muted-foreground text-xs">{p.pluginId} (plugin)</span>
                      <span className="font-mono text-xs">{formatBytes(p.size ?? 0)}</span>
                    </div>
                  ))}
                  <div className="border-muted/50 flex items-center justify-between border-t pt-1">
                    <span className="text-muted-foreground text-xs font-medium">Total</span>
                    <span className="font-mono text-xs font-medium">{formatBytes(totalBytes)}</span>
                  </div>
                </div>
              </div>
            </>
          )}

          <Divider className="my-5" />

          {/* Check for Updates */}
          <Button
            variant="outline"
            onClick={checkForUpdate}
            disabled={checkingForUpdate}
            className="w-full"
            aria-label="Check for Updates"
          >
            {checkingForUpdate ? (
              <>
                <Loader2Icon className="size-4 animate-spin" />
                Checking…
              </>
            ) : (
              "Check for Updates"
            )}
          </Button>
        </div>
      </DialogContent>
    </Dialog>
  );
};

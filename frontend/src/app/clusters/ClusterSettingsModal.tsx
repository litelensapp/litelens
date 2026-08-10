import {
  Button,
  Dialog,
  DialogContent,
  DialogFooter,
  DialogHeader,
  DialogTitle,
  Input,
} from "@litelens/design-system";
import { ClipboardGetText } from "@wailsjs/go/app/App";
import { config } from "@wailsjs/go/models";
import { FC, useEffect, useState } from "react";
import { useGetClusterProxy } from "./shared/hooks/data-access/useGetClusterProxy";
import { useGetContextKubeconfigPath } from "./shared/hooks/data-access/useGetContextKubeconfigPath";
import { useSaveClusterProxy } from "./shared/hooks/data-mutation/useSaveClusterProxy";

type SaveStatus = "idle" | "saving" | "saved" | "error";

function saveLabel(status: SaveStatus): string {
  if (status === "saving") return "Saving…";
  if (status === "saved") return "Saved!";
  return "Save";
}

interface ClusterSettingsModalProps {
  contextName: string | null;
  onClose: () => void;
  onSaved: (ctx: string) => void;
}

export const ClusterSettingsModal: FC<ClusterSettingsModalProps> = ({
  contextName,
  onClose,
  onSaved,
}) => {
  const [proxy, setProxy] = useState("");
  const [status, setStatus] = useState<SaveStatus>("idle");
  const [loadedContextName, setLoadedContextName] = useState<string | null>(null);

  const { data: kubeconfigPath } = useGetContextKubeconfigPath(contextName);
  const { data: clusterProxy } = useGetClusterProxy(contextName);
  const { mutate: saveClusterProxy } = useSaveClusterProxy();

  if (contextName && clusterProxy && contextName !== loadedContextName) {
    setLoadedContextName(contextName);
    setProxy(clusterProxy.httpProxy ?? "");
    setStatus("idle");
  } else if (!contextName && loadedContextName !== null) {
    setLoadedContextName(null);
  }

  useEffect(() => {
    if (status !== "saved") return;
    const t = setTimeout(() => setStatus("idle"), 2000);
    return () => clearTimeout(t);
  }, [status]);

  function handleSubmit(e: React.FormEvent<HTMLFormElement>) {
    e.preventDefault();
    if (!contextName) return;
    const trimmed = proxy.trim();
    setProxy(trimmed);
    setStatus("saving");
    saveClusterProxy(
      {
        contextName,
        proxy: config.ClusterProxy.createFrom({ httpProxy: trimmed, httpsProxy: trimmed }),
      },
      {
        onSuccess: () => {
          setStatus("saved");
          onSaved(contextName);
        },
        onError: () => setStatus("error"),
      }
    );
  }

  function handlePaste(e: React.ClipboardEvent<HTMLInputElement>) {
    e.preventDefault();
    setProxy(e.clipboardData.getData("text").trim());
  }

  function handleKeyDown(e: React.KeyboardEvent<HTMLInputElement>) {
    if ((e.ctrlKey || e.metaKey) && e.key === "v") {
      e.preventDefault();
      ClipboardGetText().then((text) => setProxy(text.trim()));
    }
  }

  return (
    <Dialog open={!!contextName} onOpenChange={(open) => !open && onClose()}>
      <DialogContent className="sm:max-w-md">
        <DialogHeader>
          <DialogTitle className="text-left">Cluster Settings — {contextName}</DialogTitle>
        </DialogHeader>

        <form onSubmit={handleSubmit}>
          <div className="flex flex-col gap-4 py-2">
            <div className="flex flex-col gap-1">
              <span className="text-muted-foreground text-left text-xs font-semibold uppercase tracking-wider">
                Kubeconfig
              </span>
              <p className="text-foreground truncate font-mono text-xs" title={kubeconfigPath}>
                {kubeconfigPath || "-"}
              </p>
            </div>

            <div className="flex flex-col gap-2">
              <label
                htmlFor="cluster-proxy"
                className="text-muted-foreground text-left text-xs font-semibold uppercase tracking-wider"
              >
                HTTP Proxy
              </label>
              <Input
                id="cluster-proxy"
                value={proxy}
                onChange={(e) => setProxy(e.target.value)}
                onPaste={handlePaste}
                onKeyDown={handleKeyDown}
                placeholder="http://proxy.example.com:8080"
                className="font-mono"
              />
            </div>
            <p className="text-muted-foreground text-left text-xs">
              Applied to both HTTP and HTTPS traffic. Takes effect on the next connection.
            </p>
            {status === "error" && (
              <p className="text-destructive text-xs">Failed to save. Please try again.</p>
            )}
          </div>

          <DialogFooter>
            <Button type="submit" disabled={status === "saving"}>
              {saveLabel(status)}
            </Button>
          </DialogFooter>
        </form>
      </DialogContent>
    </Dialog>
  );
};

import {
  Button,
  Dialog,
  DialogContent,
  DialogFooter,
  DialogHeader,
  DialogTitle,
  Input,
  Trash2Icon,
} from "@litelens/design-system";
import { ClipboardGetText } from "@wailsjs/go/app/App";
import { config } from "@wailsjs/go/models";
import { FC, useState } from "react";
import { NamespaceMultiSelect } from "./shared/components/NamespaceMultiSelect";
import { useGetClusterProxy } from "./shared/hooks/data-access/useGetClusterProxy";
import { useGetContextKubeconfigPath } from "./shared/hooks/data-access/useGetContextKubeconfigPath";
import { useGetDefaultNamespaces } from "./shared/hooks/data-access/useGetDefaultNamespaces";
import { useGetNamespacesForContext } from "./shared/hooks/data-access/useGetNamespacesForContext";
import { useSaveClusterProxy } from "./shared/hooks/data-mutation/useSaveClusterProxy";
import { useSaveDefaultNamespaces } from "./shared/hooks/data-mutation/useSaveDefaultNamespaces";

type SaveStatus = "idle" | "saving" | "error";

function saveLabel(status: SaveStatus): string {
  if (status === "saving") return "Saving…";
  return "Save";
}

interface ClusterSettingsModalProps {
  contextName: string | null;
  onClose: () => void;
}

export const ClusterSettingsModal: FC<ClusterSettingsModalProps> = ({ contextName, onClose }) => {
  const [proxy, setProxy] = useState("");
  const [selectedNamespaces, setSelectedNamespaces] = useState<string[]>([]);
  const [status, setStatus] = useState<SaveStatus>("idle");
  const [loadedContextName, setLoadedContextName] = useState<string | null>(null);

  const { data: kubeconfigPath } = useGetContextKubeconfigPath(contextName);
  const { data: clusterProxy } = useGetClusterProxy(contextName);
  const { data: defaultNamespaces } = useGetDefaultNamespaces(contextName);
  const {
    data: availableNamespaces,
    isLoading: isLoadingNamespaces,
    error: namespacesError,
  } = useGetNamespacesForContext(contextName);
  const { mutate: saveClusterProxy } = useSaveClusterProxy();
  const { mutate: saveDefaultNamespaces } = useSaveDefaultNamespaces();

  if (contextName && clusterProxy && contextName !== loadedContextName) {
    setLoadedContextName(contextName);
    setProxy(clusterProxy.httpProxy ?? "");
    setSelectedNamespaces(defaultNamespaces ?? []);
    setStatus("idle");
  } else if (!contextName && loadedContextName !== null) {
    setLoadedContextName(null);
  }

  const removeNamespace = (ns: string) => {
    setSelectedNamespaces((prev) => prev.filter((n) => n !== ns));
  };

  function handleSubmit(e: React.FormEvent<HTMLFormElement>) {
    e.preventDefault();
    if (!contextName) return;
    const trimmed = proxy.trim();
    setProxy(trimmed);
    setStatus("saving");

    let completed = 0;
    let hasError = false;

    const onComplete = () => {
      completed++;
      if (completed === 2) {
        if (hasError) {
          setStatus("error");
        } else {
          onClose();
        }
      }
    };

    saveClusterProxy(
      {
        contextName,
        proxy: config.ClusterProxy.createFrom({ httpProxy: trimmed, httpsProxy: trimmed }),
      },
      {
        onSuccess: onComplete,
        onError: () => {
          hasError = true;
          onComplete();
        },
      }
    );

    saveDefaultNamespaces(
      {
        contextName,
        namespaces: selectedNamespaces,
      },
      {
        onSuccess: onComplete,
        onError: () => {
          hasError = true;
          onComplete();
        },
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

            <div className="flex flex-col gap-2">
              <label className="text-muted-foreground text-left text-xs font-semibold uppercase tracking-wider">
                Default Namespaces
              </label>
              <NamespaceMultiSelect
                namespaces={selectedNamespaces}
                availableNamespaces={availableNamespaces ?? []}
                onNamespacesChange={setSelectedNamespaces}
                disabled={isLoadingNamespaces || !!namespacesError}
              />
              {namespacesError && (
                <p className="text-destructive text-xs">
                  Failed to load namespaces. Please try again.
                </p>
              )}

              {selectedNamespaces.length > 0 && (
                <div className="bg-muted/40 flex flex-col gap-1 rounded-md p-1">
                  {selectedNamespaces
                    .slice()
                    .sort()
                    .map((ns) => (
                      <div
                        key={ns}
                        className="flex items-center justify-between gap-2 rounded px-2 py-1.5"
                      >
                        <span className="text-foreground truncate text-sm">{ns}</span>
                        <Button
                          type="button"
                          variant="ghost"
                          size="icon-sm"
                          aria-label={`Remove ${ns} from default namespaces`}
                          onClick={() => removeNamespace(ns)}
                        >
                          <Trash2Icon className="h-4 w-4" />
                        </Button>
                      </div>
                    ))}
                </div>
              )}

              <p className="text-muted-foreground text-left text-xs">
                Select namespaces to use as the default filter when connecting to this context.
                Leave empty for all namespaces.
              </p>
            </div>

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

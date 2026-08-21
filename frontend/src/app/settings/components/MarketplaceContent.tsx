import {
  Button,
  CheckCircle2Icon,
  ConfirmationModal,
  Divider,
  EyeIcon,
  EyeOffIcon,
  FolderSyncIcon,
  Input,
  KeyIcon,
  LockIcon,
  LockOpenIcon,
  SaveIcon,
  Switch,
  cn,
  renderErrorToast,
  renderSuccessToast,
  XIcon,
} from "@litelens/design-system";
import { FC, useEffect, useRef, useState } from "react";
import type { config } from "@wailsjs/go/models";
import { useGetSettings } from "../hooks/data-access/useGetSettings";
import { useSectionSaveState, saveLabel } from "../hooks/useSectionSaveState";
import { usePickPluginsDir } from "../hooks/data-mutation/usePickPluginsDir";
import { useSavePluginsDir } from "../hooks/data-mutation/useSavePluginsDir";
import { useSaveMarketplaceRepositories } from "../hooks/data-mutation/useSaveMarketplaceRepositories";
import { TokenModal } from "./TokenModal";

interface MarketplaceRow {
  id: number;
  url: string;
  private: boolean;
  tokenSaved: string;
  tokenPending: string;
  tokenReplacing: boolean;
  locked: boolean;
  disabled: boolean;
}

export const MarketplaceContent: FC = () => {
  const { data: settings, isFetching: isFetchingSettings } = useGetSettings();
  const { mutateAsync: savePluginsDir } = useSavePluginsDir();
  const { mutateAsync: saveMarketplaceRepositories } = useSaveMarketplaceRepositories();
  const [pluginsDirStatus, setPluginsDirStatus] = useSectionSaveState();
  const [repoStatus, setRepoStatus] = useSectionSaveState();
  const { mutateAsync: pickPluginsDir } = usePickPluginsDir();

  const [pluginsDir, setPluginsDir] = useState("");
  const [rows, setRows] = useState<MarketplaceRow[]>([]);
  const [nextRowId, setNextRowId] = useState(1);
  const [editingRowId, setEditingRowId] = useState<number | null>(null);
  const [tokenModalOpen, setTokenModalOpen] = useState(false);
  const [unlockRowId, setUnlockRowId] = useState<number | null>(null);

  const initializedRef = useRef(false);
  const inputRefs = useRef<Map<number, HTMLInputElement>>(new Map());

  useEffect(() => {
    if (!settings || isFetchingSettings || initializedRef.current) return;
    initializedRef.current = true;
    setPluginsDir(settings.pluginsDir ?? "");

    // Initialize rows from settings.marketplaceRepositories
    const initialRows: MarketplaceRow[] = (settings.marketplaceRepositories ?? []).map(
      (repo, idx) => ({
        id: idx + 1,
        url: repo.url,
        private: repo.private,
        tokenSaved: repo.accessToken ?? "",
        tokenPending: "",
        tokenReplacing: false,
        locked: repo.locked ?? false,
        disabled: repo.disabled ?? false,
      })
    );
    setRows(initialRows);
    setNextRowId((initialRows.length || 0) + 1);
  }, [settings, isFetchingSettings]);

  async function handleBrowse() {
    const picked = await pickPluginsDir().catch(() => "");
    if (!picked) return;
    initializedRef.current = true;
    setPluginsDir(picked);
  }

  function handleUrlChange(rowId: number, value: string) {
    initializedRef.current = true;
    setRows((prev) => prev.map((r) => (r.id === rowId ? { ...r, url: value } : r)));
  }

  function handlePrivateChange(rowId: number, checked: boolean) {
    initializedRef.current = true;
    setRows((prev) => prev.map((r) => (r.id === rowId ? { ...r, private: checked } : r)));
  }

  function handleTokenModalOpen(rowId: number) {
    const row = rows.find((r) => r.id === rowId);
    if (!row) return;
    setEditingRowId(rowId);
    if (row.tokenSaved) {
      setRows((prev) =>
        prev.map((r) =>
          r.id === rowId ? { ...r, tokenReplacing: true, tokenPending: row.tokenSaved } : r
        )
      );
    }
    setTokenModalOpen(true);
  }

  function handleTokenModalClose() {
    setTokenModalOpen(false);
    setEditingRowId(null);
    setRows((prev) =>
      prev.map((r) =>
        r.id === editingRowId ? { ...r, tokenReplacing: false, tokenPending: "" } : r
      )
    );
  }

  function handleTokenModalSubmit(token: string) {
    initializedRef.current = true;
    if (editingRowId !== null) {
      setRows((prev) =>
        prev.map((r) => (r.id === editingRowId ? { ...r, tokenPending: token } : r))
      );
    }
    setTokenModalOpen(false);
  }

  function handleRemoveRow(rowId: number) {
    initializedRef.current = true;
    setRows((prev) => prev.filter((r) => r.id !== rowId));
  }

  function buildMarketplaceRepositoriesPayload(
    rowsList: MarketplaceRow[]
  ): config.MarketplaceRepository[] {
    return rowsList
      .filter((r) => r.url.trim())
      .map((r) => ({
        url: r.url,
        private: r.private,
        accessToken: r.tokenReplacing || !r.tokenSaved ? r.tokenPending : r.tokenSaved,
        locked: r.locked,
        disabled: r.disabled,
      }));
  }

  async function persistLockChange(rowId: number, locked: boolean) {
    initializedRef.current = true;
    const nextRows = rows.map((r) => (r.id === rowId ? { ...r, locked } : r));
    setRows(nextRows);
    try {
      await saveMarketplaceRepositories(buildMarketplaceRepositoriesPayload(nextRows));
    } catch {
      // Revert the optimistic UI update if persisting the lock state failed.
      setRows((prev) => prev.map((r) => (r.id === rowId ? { ...r, locked: !locked } : r)));
      renderErrorToast({
        title: "Failed to update lock state",
        description: "Please try again.",
      });
    }
  }

  async function persistDisableChange(rowId: number, disabled: boolean) {
    initializedRef.current = true;
    const nextRows = rows.map((r) => (r.id === rowId ? { ...r, disabled } : r));
    setRows(nextRows);
    try {
      await saveMarketplaceRepositories(buildMarketplaceRepositoriesPayload(nextRows));
    } catch {
      // Revert the optimistic UI update if persisting the disabled state failed.
      setRows((prev) => prev.map((r) => (r.id === rowId ? { ...r, disabled: !disabled } : r)));
      renderErrorToast({
        title: "Failed to update disable state",
        description: "Please try again.",
      });
    }
  }

  function handleToggleLock(rowId: number) {
    const row = rows.find((r) => r.id === rowId);
    if (!row) return;
    if (row.locked) {
      setUnlockRowId(rowId);
      return;
    }
    void persistLockChange(rowId, true);
  }

  function handleUnlockConfirm() {
    if (unlockRowId === null) return;
    void persistLockChange(unlockRowId, false);
    setUnlockRowId(null);
  }

  function handleUnlockCancel() {
    setUnlockRowId(null);
  }

  function handleToggleDisable(rowId: number) {
    const row = rows.find((r) => r.id === rowId);
    if (!row) return;
    void persistDisableChange(rowId, !row.disabled);
  }

  function handleAddRow() {
    initializedRef.current = true;
    const newRow: MarketplaceRow = {
      id: nextRowId,
      url: "",
      private: false,
      tokenSaved: "",
      tokenPending: "",
      tokenReplacing: false,
      locked: false,
      disabled: false,
    };
    setRows((prev) => [...prev, newRow]);
    setNextRowId((prev) => prev + 1);
    // Focus new row's input
    setTimeout(() => {
      const input = inputRefs.current.get(newRow.id);
      if (input) input.focus();
    }, 0);
  }

  async function handleSavePluginsDir() {
    setPluginsDirStatus("saving");
    try {
      await savePluginsDir(pluginsDir);
      setPluginsDirStatus("saved");
      renderSuccessToast({
        title: "Marketplace settings saved",
        description: "Plugins directory configuration has been updated.",
      });
    } catch {
      setPluginsDirStatus("error");
    }
  }

  function handleMarketplaceFormSubmit(e: React.FormEvent<HTMLFormElement>) {
    e.preventDefault();
    handleSaveMarketplaceRepo();
  }

  async function handleSaveMarketplaceRepo() {
    setRepoStatus("saving");
    try {
      await saveMarketplaceRepositories(buildMarketplaceRepositoriesPayload(rows));

      // Update tokenSaved from resolved values
      setRows((prev) =>
        prev.map((r) => ({
          ...r,
          tokenSaved: r.tokenReplacing || !r.tokenSaved ? r.tokenPending : r.tokenSaved,
          tokenPending: "",
          tokenReplacing: false,
        }))
      );

      setRepoStatus("saved");
      renderSuccessToast({
        title: "Marketplace settings saved",
        description: "Marketplace repository configuration has been updated.",
      });
    } catch {
      setRepoStatus("error");
    }
  }

  return (
    <div className="flex flex-1 flex-col overflow-hidden">
      <div className="flex flex-1 flex-col gap-6 overflow-hidden">
        <div className="flex min-h-0 flex-1 flex-col gap-6">
          <div className="flex max-w-3xl flex-col gap-2">
            <p className="text-left text-xs font-semibold uppercase tracking-wider">
              Plugins Directory
            </p>
            <div className="flex items-end gap-2">
              <Input
                value={pluginsDir}
                onChange={(e) => {
                  initializedRef.current = true;
                  setPluginsDir(e.target.value);
                }}
                placeholder="~/.litelens/plugins"
                aria-label="Plugins directory path"
                className="flex-1 font-mono text-sm"
              />
              <Button onClick={handleBrowse} className="w-fit">
                <FolderSyncIcon className="size-4" />
                Browse
              </Button>
              <Button
                onClick={handleSavePluginsDir}
                disabled={pluginsDirStatus === "saving" || !settings}
                className="w-fit"
              >
                <SaveIcon className="size-3.5" />
                {saveLabel(pluginsDirStatus)}
              </Button>
            </div>
            {pluginsDirStatus === "error" && (
              <p className="text-destructive text-xs">Failed to save. Please try again.</p>
            )}
          </div>

          <Divider />

          <form
            className="flex min-h-0 max-w-4xl flex-1 flex-col gap-4"
            onSubmit={handleMarketplaceFormSubmit}
          >
            <div className="flex items-center gap-4">
              <p className="text-left text-xs font-semibold uppercase tracking-wider">
                Marketplace Repository URLs
              </p>
              <Button type="submit" size="sm" disabled={repoStatus === "saving" || !settings}>
                <SaveIcon className="size-3.5" />
                {saveLabel(repoStatus)}
              </Button>
              <Button type="button" variant="ghost" size="sm" onClick={handleAddRow}>
                + Add Marketplace
              </Button>
            </div>

            {repoStatus === "error" && (
              <p className="text-destructive text-xs">Failed to save. Please try again.</p>
            )}

            <p className="text-muted-foreground text-xs">
              Enter any marketplace source URL. The official litelens marketplace is pre-added below
              — remove it if you don't want plugins fetched from it.
            </p>

            {/* User-added repositories */}
            {rows.length > 0 && (
              <div className="flex min-h-0 flex-1 flex-col gap-4 overflow-y-auto pr-1">
                {rows.map((row) => (
                  <div
                    key={row.id}
                    className={cn(
                      "flex flex-col gap-3 rounded-md border p-3",
                      row.disabled && "opacity-60"
                    )}
                  >
                    {/* URL input + token button + private switch + remove button */}
                    <div className="flex items-center gap-4">
                      <Input
                        ref={(el) => {
                          if (el) inputRefs.current.set(row.id, el);
                        }}
                        value={row.url}
                        onChange={(e) => handleUrlChange(row.id, e.target.value)}
                        placeholder="https://github.com/user/plugins"
                        aria-label={`Marketplace repository URL ${row.id}`}
                        className="flex-1 font-mono text-sm"
                        disabled={row.locked}
                      />

                      {row.private && (
                        <Button
                          variant="ghost"
                          size="icon-sm"
                          onClick={() => handleTokenModalOpen(row.id)}
                          aria-label={row.tokenSaved ? "Update access token" : "Add access token"}
                          className="shrink-0"
                          disabled={row.locked}
                        >
                          <div className="flex items-center gap-1">
                            <KeyIcon className="text-muted-foreground size-4" />
                            {row.tokenSaved && (
                              <CheckCircle2Icon className="size-3 text-green-500" />
                            )}
                          </div>
                        </Button>
                      )}

                      <div className="flex shrink-0 items-center gap-1.5">
                        <label htmlFor={`private-${row.id}`} className="text-xs font-medium">
                          Private
                        </label>
                        <Switch
                          id={`private-${row.id}`}
                          checked={row.private}
                          onCheckedChange={(checked) => handlePrivateChange(row.id, checked)}
                          aria-label={`Mark marketplace repository ${row.id} as private`}
                          disabled={row.locked}
                        />
                      </div>

                      <Button
                        variant="ghost"
                        size="icon-sm"
                        onClick={() => handleRemoveRow(row.id)}
                        aria-label="Remove this marketplace repository"
                        className="shrink-0"
                        disabled={row.locked}
                      >
                        <XIcon className="size-4" />
                      </Button>

                      <Button
                        variant="ghost"
                        size="icon-sm"
                        onClick={() => handleToggleDisable(row.id)}
                        aria-label={
                          row.disabled
                            ? "Enable this marketplace repository"
                            : "Disable this marketplace repository"
                        }
                        aria-pressed={row.disabled}
                        className="shrink-0"
                      >
                        {row.disabled ? (
                          <EyeOffIcon className="size-4" />
                        ) : (
                          <EyeIcon className="size-4" />
                        )}
                      </Button>

                      <Button
                        variant="ghost"
                        size="icon-sm"
                        onClick={() => handleToggleLock(row.id)}
                        aria-label={
                          row.locked
                            ? "Unlock this marketplace repository"
                            : "Lock this marketplace repository"
                        }
                        aria-pressed={row.locked}
                        className="shrink-0"
                      >
                        {row.locked ? (
                          <LockIcon className="size-4" />
                        ) : (
                          <LockOpenIcon className="text-muted-foreground size-4" />
                        )}
                      </Button>
                    </div>
                  </div>
                ))}
              </div>
            )}
          </form>

          <TokenModal
            open={tokenModalOpen}
            onClose={handleTokenModalClose}
            onSubmit={handleTokenModalSubmit}
            savedToken={
              editingRowId !== null
                ? (rows.find((r) => r.id === editingRowId)?.tokenSaved ?? "")
                : ""
            }
          />

          <ConfirmationModal
            open={unlockRowId !== null}
            title="Unlock repository"
            description="This will allow this marketplace repository's fields to be edited again. Are you sure you want to unlock it?"
            confirmLabel="Unlock"
            isPending={false}
            onClose={handleUnlockCancel}
            onConfirm={handleUnlockConfirm}
          />
        </div>
      </div>
    </div>
  );
};

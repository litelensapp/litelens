import {
  Button,
  Input,
  renderSuccessToast,
  SaveIcon,
  TimezoneSelect,
} from "@litelens/design-system";
import { FC, useEffect, useRef, useState } from "react";
import { useGetDefaultShell } from "../hooks/data-access/useGetDefaultShell";
import { useGetSettings } from "../hooks/data-access/useGetSettings";
import { useSaveLocaleTimezone } from "../hooks/data-mutation/useSaveLocaleTimezone";
import { useMergeSettingsOnSave } from "../hooks/useMergeSettingsOnSave";
import { saveLabel, useSectionSaveState } from "../hooks/useSectionSaveState";

const SYSTEM_TZ = Intl.DateTimeFormat().resolvedOptions().timeZone;

export const AppContent: FC = () => {
  const { data: settings } = useGetSettings();
  const { mutate: saveTimezone } = useSaveLocaleTimezone();
  const mergeAndSave = useMergeSettingsOnSave();
  const [shellPathStatus, setShellPathStatus] = useSectionSaveState();

  const { data: defaultShell = "/bin/zsh" } = useGetDefaultShell();

  const [shellPath, setShellPath] = useState("");

  const initializedRef = useRef(false);

  const currentTz = settings?.locale || SYSTEM_TZ;

  useEffect(() => {
    if (!settings || initializedRef.current) return;
    initializedRef.current = true;
    setShellPath(settings.shellPath ?? "");
  }, [settings]);

  function handleTimezoneChange(tz: string) {
    saveTimezone(tz, {
      onSuccess: () => renderSuccessToast({ title: "Timezone updated", description: tz }),
    });
  }

  async function handleSaveShellPath() {
    setShellPathStatus("saving");
    try {
      await mergeAndSave({ shellPath });
      setShellPathStatus("saved");
      renderSuccessToast({
        title: "Terminal settings saved",
        description: "Terminal configuration has been updated.",
      });
    } catch {
      setShellPathStatus("error");
    }
  }

  return (
    <div className="flex flex-col gap-6">
      <div className="flex max-w-sm flex-col gap-2">
        <div id="tz-label" className="text-left text-xs font-semibold uppercase tracking-wider">
          Locale Timezone
        </div>
        <TimezoneSelect
          value={currentTz}
          onChange={handleTimezoneChange}
          aria-labelledby="tz-label"
        />
      </div>

      <div className="flex max-w-md flex-col gap-2">
        <label
          className="text-left text-xs font-semibold uppercase tracking-wider"
          htmlFor="shell-path"
        >
          Terminal Shell Path
        </label>
        <div className="flex items-center gap-2">
          <Input
            id="shell-path"
            value={shellPath}
            onChange={(e) => setShellPath(e.target.value)}
            placeholder={defaultShell}
            className="flex-1 font-mono"
          />
          <Button
            size="sm"
            onClick={handleSaveShellPath}
            disabled={shellPathStatus === "saving" || !settings}
          >
            <SaveIcon className="size-3.5" />
            {saveLabel(shellPathStatus)}
          </Button>
        </div>
        {shellPathStatus === "error" && (
          <p className="text-destructive text-xs">Failed to save. Please try again.</p>
        )}
      </div>
    </div>
  );
};

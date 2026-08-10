import { useEffect, useState } from "react";
import { EventsOn } from "@wailsjs/runtime/runtime";

const DISMISSED_UPDATE_KEY = "litelens:dismissed-update";

export interface UpdateInfo {
  latestVersion: string;
  releaseURL: string;
  downloadSize: string;
}

export interface UseUpdateAvailableEventsResult {
  updateInfo: UpdateInfo | null;
  updateModalOpen: boolean;
  setUpdateModalOpen: (open: boolean) => void;
  dismissUpdate: () => void;
}

export function useUpdateAvailableEvents(): UseUpdateAvailableEventsResult {
  const [updateInfo, setUpdateInfo] = useState<UpdateInfo | null>(null);
  const [updateModalOpen, setUpdateModalOpen] = useState(false);

  useEffect(() => {
    return EventsOn(
      "update:available",
      (payload: { latestVersion: string; releaseURL: string; downloadSize: string }) => {
        const info: UpdateInfo = {
          latestVersion: payload.latestVersion,
          releaseURL: payload.releaseURL,
          downloadSize: payload.downloadSize ?? "",
        };
        setUpdateInfo(info);
        const dismissed = localStorage.getItem(DISMISSED_UPDATE_KEY);
        if (dismissed !== info.latestVersion) {
          setUpdateModalOpen(true);
        }
      }
    );
  }, []);

  const dismissUpdate = () => {
    setUpdateModalOpen(false);
    if (updateInfo) {
      localStorage.setItem(DISMISSED_UPDATE_KEY, updateInfo.latestVersion);
    }
  };

  return { updateInfo, updateModalOpen, setUpdateModalOpen, dismissUpdate };
}

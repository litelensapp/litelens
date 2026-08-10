import { CheckForUpdate } from "@wailsjs/go/app/App";
import { useState } from "react";

export const useCheckForUpdate = () => {
  const [checkingForUpdate, setCheckingForUpdate] = useState(false);

  const checkForUpdate = async () => {
    setCheckingForUpdate(true);
    try {
      await CheckForUpdate();
    } finally {
      setCheckingForUpdate(false);
    }
  };

  return { checkingForUpdate, checkForUpdate };
};

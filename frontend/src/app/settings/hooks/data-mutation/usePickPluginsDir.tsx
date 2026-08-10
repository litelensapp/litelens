import { useMutation } from "@tanstack/react-query";
import { PickPluginsDir } from "@wailsjs/go/app/App";

export const usePickPluginsDir = () => {
  return useMutation({
    mutationFn: () => PickPluginsDir(),
  });
};

import { useMutation, useQueryClient } from "@tanstack/react-query";
import { SavePluginsDir } from "@wailsjs/go/app/App";
import { QUERY_KEY_SETTINGS } from "../../api/api.const";

export const useSavePluginsDir = () => {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (dir: string) => SavePluginsDir(dir),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: [QUERY_KEY_SETTINGS] });
    },
  });
};

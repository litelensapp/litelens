import { renderErrorToast } from "@litelens/design-system";
import { useMutation } from "@tanstack/react-query";
import { SetActiveNamespaces } from "../../api/resources";

export const useSetActiveNamespaces = () => {
  return useMutation({
    mutationFn: (namespaces: string[]) => SetActiveNamespaces(namespaces),
    onError: (err) =>
      renderErrorToast({
        title: "Failed to set active namespaces",
        description: String(err),
      }),
  });
};

import { renderErrorToast } from "@litelens/design-system";
import { useMutation, useQueryClient } from "@tanstack/react-query";
import { SetActiveNamespaces } from "../../api/resources";

export const useSetActiveNamespaces = () => {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: (namespaces: string[]) => SetActiveNamespaces(namespaces),
    // The caller flips local `namespaces` state (part of every list query's
    // key) in lockstep with this call, so most queries re-fetch under a new
    // key immediately — before the backend's active-namespace state is
    // actually updated. That in-flight fetch can land under the new key with
    // data still filtered by the OLD selection. Invalidating every
    // namespace-scoped query once the backend confirms the change forces a
    // re-fetch against the now-correct backend state, closing that race.
    onSuccess: () =>
      queryClient.invalidateQueries({
        predicate: (query) =>
          query.queryKey.some((k) => typeof k === "object" && k !== null && "namespaces" in k),
      }),
    onError: (err) =>
      renderErrorToast({
        title: "Failed to set active namespaces",
        description: String(err),
      }),
  });
};

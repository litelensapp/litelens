import { useMutation } from "@tanstack/react-query";
import { Connect } from "@wailsjs/go/app/App";

// Module-scoped (not per-hook-instance) so every call site shares one
// monotonic counter — there's only one logical "active context" on the
// backend to order calls against.
let callSeq = 0;

export const useConnect = () =>
  useMutation({
    // Rapid back-and-forth cluster switches each fire their own untracked,
    // unordered mutate() call, and each does slow network/informer-sync work
    // before the backend commits it as active — so they can finish in an
    // order that doesn't match the order the user clicked them in. Passing a
    // synchronously-incremented seq lets the backend drop a call that's been
    // superseded by a newer one, instead of a stale switch silently winning
    // and leaving the backend's active context out of sync with the UI. See
    // App.Connect.
    mutationFn: (ctx: string) => Connect(ctx, ++callSeq),
  });

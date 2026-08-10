import { useMutation } from "@tanstack/react-query";
import { Connect } from "@wailsjs/go/app/App";

export const useConnect = () =>
  useMutation({
    mutationFn: (ctx: string) => Connect(ctx),
  });

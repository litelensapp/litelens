import { renderErrorToast } from "@litelens/design-system";
import { useMutation } from "@tanstack/react-query";
import { DownloadPodLogs } from "../api/resources";

interface DownloadPodLogsVariables {
  contextName: string;
  ns: string;
  pod: string;
  container: string;
}

export const useDownloadPodLogs = () =>
  useMutation({
    mutationFn: ({ contextName, ns, pod, container }: DownloadPodLogsVariables) =>
      DownloadPodLogs(contextName, ns, pod, container),
    onError: (err) => {
      renderErrorToast({ title: "Failed to download logs", description: String(err) });
    },
  });

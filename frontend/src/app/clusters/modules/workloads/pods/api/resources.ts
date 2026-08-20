export {
  ExecInPod,
  GetPodByName,
  GetPodsSummary,
  GetPodYAML,
  ListPods,
  ResizeExecTerminal,
  StopExec,
  StopLogs,
  StreamLogs,
  UpdatePodYAML,
} from "@wailsjs/go/app/App";

export type {
  PodCondition,
  PodContainerPort,
  PodContainerMount,
  PodContainerLastStatus,
  PodContainerDetail,
  PodVolumeSource,
  PodVolume,
  TolerationDetail,
  Pod,
  PodSummary,
} from "@litelens/core";

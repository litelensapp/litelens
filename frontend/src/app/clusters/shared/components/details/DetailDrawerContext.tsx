import { createContext, FC, ReactNode, use, useMemo, useReducer } from "react";

interface DetailDrawerContextValue {
  selectedNamespaceName: string | null;
  onToggleNamespaceDetail: (name?: string) => void;

  selectedClusterRoleName: string | null;
  onToggleClusterRoleDetail: (name?: string) => void;

  selectedClusterRoleBindingName: string | null;
  onToggleClusterRoleBindingDetail: (name?: string) => void;

  selectedIngressClassName: string | null;
  onToggleIngressClassDetail: (name?: string) => void;

  selectedValidatingWebhookConfigName: string | null;
  onToggleValidatingWebhookConfigDetail: (name?: string) => void;

  selectedRoleName: string | null;
  selectedRoleNamespace: string | null;
  onToggleRoleDetail: (namespace?: string, name?: string) => void;

  selectedRoleBindingName: string | null;
  selectedRoleBindingNamespace: string | null;
  onToggleRoleBindingDetail: (namespace?: string, name?: string) => void;

  selectedServiceAccountName: string | null;
  selectedServiceAccountNamespace: string | null;
  onToggleServiceAccountDetail: (namespace?: string, name?: string) => void;

  selectedPodName: string | null;
  selectedPodNamespace: string | null;
  onTogglePodDetail: (namespace?: string, name?: string) => void;

  selectedJobName: string | null;
  selectedJobNamespace: string | null;
  onToggleJobDetail: (namespace?: string, name?: string) => void;

  selectedCronJobName: string | null;
  selectedCronJobNamespace: string | null;
  onToggleCronJobDetail: (namespace?: string, name?: string) => void;

  selectedNodeName: string | null;
  onToggleNodeDetail: (name?: string) => void;

  selectedServiceName: string | null;
  selectedServiceNamespace: string | null;
  onToggleServiceDetail: (namespace?: string, name?: string) => void;

  selectedDeploymentName: string | null;
  selectedDeploymentNamespace: string | null;
  onToggleDeploymentDetail: (namespace?: string, name?: string) => void;

  selectedReplicaSetName: string | null;
  selectedReplicaSetNamespace: string | null;
  onToggleReplicaSetDetail: (namespace?: string, name?: string) => void;

  selectedDaemonSetName: string | null;
  selectedDaemonSetNamespace: string | null;
  onToggleDaemonSetDetail: (namespace?: string, name?: string) => void;

  selectedStatefulSetName: string | null;
  selectedStatefulSetNamespace: string | null;
  onToggleStatefulSetDetail: (namespace?: string, name?: string) => void;

  selectedEventName: string | null;
  selectedEventNamespace: string | null;
  onToggleEventDetail: (namespace?: string, name?: string) => void;

  selectedConfigMapName: string | null;
  selectedConfigMapNamespace: string | null;
  onToggleConfigMapDetail: (namespace?: string, name?: string) => void;

  selectedNetworkPolicyName: string | null;
  selectedNetworkPolicyNamespace: string | null;
  onToggleNetworkPolicyDetail: (namespace?: string, name?: string) => void;

  selectedPersistentVolumeClaimName: string | null;
  selectedPersistentVolumeClaimNamespace: string | null;
  onTogglePersistentVolumeClaimDetail: (namespace?: string, name?: string) => void;

  selectedPodDisruptionBudgetName: string | null;
  selectedPodDisruptionBudgetNamespace: string | null;
  onTogglePodDisruptionBudgetDetail: (namespace?: string, name?: string) => void;

  selectedSecretName: string | null;
  selectedSecretNamespace: string | null;
  onToggleSecretDetail: (namespace?: string, name?: string) => void;

  selectedHPAName: string | null;
  selectedHPANamespace: string | null;
  onToggleHPADetail: (namespace?: string, name?: string) => void;

  selectedIngressName: string | null;
  selectedIngressNamespace: string | null;
  onToggleIngressDetail: (namespace?: string, name?: string) => void;

  selectedResourceQuotaName: string | null;
  selectedResourceQuotaNamespace: string | null;
  onToggleResourceQuotaDetail: (namespace?: string, name?: string) => void;

  selectedLimitRangeName: string | null;
  selectedLimitRangeNamespace: string | null;
  onToggleLimitRangeDetail: (namespace?: string, name?: string) => void;

  selectedEndpointName: string | null;
  selectedEndpointNamespace: string | null;
  onToggleEndpointDetail: (namespace?: string, name?: string) => void;

  selectedEndpointSliceName: string | null;
  selectedEndpointSliceNamespace: string | null;
  onToggleEndpointSliceDetail: (namespace?: string, name?: string) => void;

  selectedLeaseName: string | null;
  selectedLeaseNamespace: string | null;
  onToggleLease: (namespace?: string, name?: string) => void;

  selectedPriorityClassName: string | null;
  onTogglePriorityClass: (name?: string) => void;

  selectedPersistentVolumeName: string | null;
  onTogglePersistentVolumeDetail: (name?: string) => void;

  selectedStorageClassName: string | null;
  onToggleStorageClassDetail: (name?: string) => void;
}

interface DetailDrawerState {
  selectedNamespaceName: string | null;
  selectedClusterRoleName: string | null;
  selectedClusterRoleBindingName: string | null;
  selectedIngressClassName: string | null;
  selectedValidatingWebhookConfigName: string | null;
  selectedRoleName: string | null;
  selectedRoleNamespace: string | null;
  selectedRoleBindingName: string | null;
  selectedRoleBindingNamespace: string | null;
  selectedServiceAccountName: string | null;
  selectedServiceAccountNamespace: string | null;
  selectedPodName: string | null;
  selectedPodNamespace: string | null;
  selectedJobName: string | null;
  selectedJobNamespace: string | null;
  selectedCronJobName: string | null;
  selectedCronJobNamespace: string | null;
  selectedNodeName: string | null;
  selectedServiceName: string | null;
  selectedServiceNamespace: string | null;
  selectedDeploymentName: string | null;
  selectedDeploymentNamespace: string | null;
  selectedReplicaSetName: string | null;
  selectedReplicaSetNamespace: string | null;
  selectedDaemonSetName: string | null;
  selectedDaemonSetNamespace: string | null;
  selectedStatefulSetName: string | null;
  selectedStatefulSetNamespace: string | null;
  selectedEventName: string | null;
  selectedEventNamespace: string | null;
  selectedConfigMapName: string | null;
  selectedConfigMapNamespace: string | null;
  selectedNetworkPolicyName: string | null;
  selectedNetworkPolicyNamespace: string | null;
  selectedPersistentVolumeClaimName: string | null;
  selectedPersistentVolumeClaimNamespace: string | null;
  selectedPodDisruptionBudgetName: string | null;
  selectedPodDisruptionBudgetNamespace: string | null;
  selectedSecretName: string | null;
  selectedSecretNamespace: string | null;
  selectedHPAName: string | null;
  selectedHPANamespace: string | null;
  selectedIngressName: string | null;
  selectedIngressNamespace: string | null;
  selectedResourceQuotaName: string | null;
  selectedResourceQuotaNamespace: string | null;
  selectedLimitRangeName: string | null;
  selectedLimitRangeNamespace: string | null;
  selectedEndpointName: string | null;
  selectedEndpointNamespace: string | null;
  selectedEndpointSliceName: string | null;
  selectedEndpointSliceNamespace: string | null;
  selectedLeaseName: string | null;
  selectedLeaseNamespace: string | null;
  selectedPriorityClassName: string | null;
  selectedPersistentVolumeName: string | null;
  selectedStorageClassName: string | null;
}

type DetailDrawerAction =
  | { type: "toggleNamespace"; name?: string }
  | { type: "toggleClusterRole"; name?: string }
  | { type: "toggleClusterRoleBinding"; name?: string }
  | { type: "toggleIngressClass"; name?: string }
  | { type: "toggleValidatingWebhookConfig"; name?: string }
  | { type: "toggleRole"; namespace?: string; name?: string }
  | { type: "toggleRoleBinding"; namespace?: string; name?: string }
  | { type: "toggleServiceAccount"; namespace?: string; name?: string }
  | { type: "togglePod"; namespace?: string; name?: string }
  | { type: "toggleJob"; namespace?: string; name?: string }
  | { type: "toggleCronJob"; namespace?: string; name?: string }
  | { type: "toggleNode"; name?: string }
  | { type: "toggleService"; namespace?: string; name?: string }
  | { type: "toggleDeployment"; namespace?: string; name?: string }
  | { type: "toggleReplicaSet"; namespace?: string; name?: string }
  | { type: "toggleDaemonSet"; namespace?: string; name?: string }
  | { type: "toggleStatefulSet"; namespace?: string; name?: string }
  | { type: "toggleEvent"; namespace?: string; name?: string }
  | { type: "toggleConfigMap"; namespace?: string; name?: string }
  | { type: "toggleNetworkPolicy"; namespace?: string; name?: string }
  | { type: "togglePersistentVolumeClaim"; namespace?: string; name?: string }
  | { type: "togglePodDisruptionBudget"; namespace?: string; name?: string }
  | { type: "toggleSecret"; namespace?: string; name?: string }
  | { type: "toggleHPA"; namespace?: string; name?: string }
  | { type: "toggleIngress"; namespace?: string; name?: string }
  | { type: "toggleResourceQuota"; namespace?: string; name?: string }
  | { type: "toggleLimitRange"; namespace?: string; name?: string }
  | { type: "toggleEndpoint"; namespace?: string; name?: string }
  | { type: "toggleEndpointSlice"; namespace?: string; name?: string }
  | { type: "toggleLease"; namespace?: string; name?: string }
  | { type: "togglePriorityClass"; name?: string }
  | { type: "togglePersistentVolume"; name?: string }
  | { type: "toggleStorageClass"; name?: string };

const initialState: DetailDrawerState = {
  selectedNamespaceName: null,
  selectedClusterRoleName: null,
  selectedClusterRoleBindingName: null,
  selectedIngressClassName: null,
  selectedValidatingWebhookConfigName: null,
  selectedRoleName: null,
  selectedRoleNamespace: null,
  selectedRoleBindingName: null,
  selectedRoleBindingNamespace: null,
  selectedServiceAccountName: null,
  selectedServiceAccountNamespace: null,
  selectedPodName: null,
  selectedPodNamespace: null,
  selectedJobName: null,
  selectedJobNamespace: null,
  selectedCronJobName: null,
  selectedCronJobNamespace: null,
  selectedNodeName: null,
  selectedServiceName: null,
  selectedServiceNamespace: null,
  selectedDeploymentName: null,
  selectedDeploymentNamespace: null,
  selectedReplicaSetName: null,
  selectedReplicaSetNamespace: null,
  selectedDaemonSetName: null,
  selectedDaemonSetNamespace: null,
  selectedStatefulSetName: null,
  selectedStatefulSetNamespace: null,
  selectedEventName: null,
  selectedEventNamespace: null,
  selectedConfigMapName: null,
  selectedConfigMapNamespace: null,
  selectedNetworkPolicyName: null,
  selectedNetworkPolicyNamespace: null,
  selectedPersistentVolumeClaimName: null,
  selectedPersistentVolumeClaimNamespace: null,
  selectedPodDisruptionBudgetName: null,
  selectedPodDisruptionBudgetNamespace: null,
  selectedSecretName: null,
  selectedSecretNamespace: null,
  selectedHPAName: null,
  selectedHPANamespace: null,
  selectedIngressName: null,
  selectedIngressNamespace: null,
  selectedResourceQuotaName: null,
  selectedResourceQuotaNamespace: null,
  selectedLimitRangeName: null,
  selectedLimitRangeNamespace: null,
  selectedEndpointName: null,
  selectedEndpointNamespace: null,
  selectedEndpointSliceName: null,
  selectedEndpointSliceNamespace: null,
  selectedLeaseName: null,
  selectedLeaseNamespace: null,
  selectedPriorityClassName: null,
  selectedPersistentVolumeName: null,
  selectedStorageClassName: null,
};

function detailDrawerReducer(
  state: DetailDrawerState,
  action: DetailDrawerAction
): DetailDrawerState {
  switch (action.type) {
    case "toggleNamespace":
      return { ...state, selectedNamespaceName: action.name ?? null };
    case "toggleClusterRole":
      return { ...state, selectedClusterRoleName: action.name ?? null };
    case "toggleClusterRoleBinding":
      return { ...state, selectedClusterRoleBindingName: action.name ?? null };
    case "toggleIngressClass":
      return { ...state, selectedIngressClassName: action.name ?? null };
    case "toggleValidatingWebhookConfig":
      return { ...state, selectedValidatingWebhookConfigName: action.name ?? null };
    case "toggleRole":
      return {
        ...state,
        selectedRoleNamespace: action.namespace ?? null,
        selectedRoleName: action.name ?? null,
      };
    case "toggleRoleBinding":
      return {
        ...state,
        selectedRoleBindingNamespace: action.namespace ?? null,
        selectedRoleBindingName: action.name ?? null,
      };
    case "toggleServiceAccount":
      return {
        ...state,
        selectedServiceAccountNamespace: action.namespace ?? null,
        selectedServiceAccountName: action.name ?? null,
      };
    case "togglePod":
      return {
        ...state,
        selectedPodNamespace: action.namespace ?? null,
        selectedPodName: action.name ?? null,
      };
    case "toggleJob":
      return {
        ...state,
        selectedJobNamespace: action.namespace ?? null,
        selectedJobName: action.name ?? null,
      };
    case "toggleCronJob":
      return {
        ...state,
        selectedCronJobNamespace: action.namespace ?? null,
        selectedCronJobName: action.name ?? null,
      };
    case "toggleNode":
      return { ...state, selectedNodeName: action.name ?? null };
    case "toggleService":
      return {
        ...state,
        selectedServiceNamespace: action.namespace ?? null,
        selectedServiceName: action.name ?? null,
      };
    case "toggleDeployment":
      return {
        ...state,
        selectedDeploymentNamespace: action.namespace ?? null,
        selectedDeploymentName: action.name ?? null,
      };
    case "toggleReplicaSet":
      return {
        ...state,
        selectedReplicaSetNamespace: action.namespace ?? null,
        selectedReplicaSetName: action.name ?? null,
      };
    case "toggleDaemonSet":
      return {
        ...state,
        selectedDaemonSetNamespace: action.namespace ?? null,
        selectedDaemonSetName: action.name ?? null,
      };
    case "toggleStatefulSet":
      return {
        ...state,
        selectedStatefulSetNamespace: action.namespace ?? null,
        selectedStatefulSetName: action.name ?? null,
      };
    case "toggleEvent":
      return {
        ...state,
        selectedEventNamespace: action.namespace ?? null,
        selectedEventName: action.name ?? null,
      };
    case "toggleConfigMap":
      return {
        ...state,
        selectedConfigMapNamespace: action.namespace ?? null,
        selectedConfigMapName: action.name ?? null,
      };
    case "toggleNetworkPolicy":
      return {
        ...state,
        selectedNetworkPolicyNamespace: action.namespace ?? null,
        selectedNetworkPolicyName: action.name ?? null,
      };
    case "togglePersistentVolumeClaim":
      return {
        ...state,
        selectedPersistentVolumeClaimNamespace: action.namespace ?? null,
        selectedPersistentVolumeClaimName: action.name ?? null,
      };
    case "togglePodDisruptionBudget":
      return {
        ...state,
        selectedPodDisruptionBudgetNamespace: action.namespace ?? null,
        selectedPodDisruptionBudgetName: action.name ?? null,
      };
    case "toggleSecret":
      return {
        ...state,
        selectedSecretNamespace: action.namespace ?? null,
        selectedSecretName: action.name ?? null,
      };
    case "toggleHPA":
      return {
        ...state,
        selectedHPANamespace: action.namespace ?? null,
        selectedHPAName: action.name ?? null,
      };
    case "toggleIngress":
      return {
        ...state,
        selectedIngressNamespace: action.namespace ?? null,
        selectedIngressName: action.name ?? null,
      };
    case "toggleResourceQuota":
      return {
        ...state,
        selectedResourceQuotaNamespace: action.namespace ?? null,
        selectedResourceQuotaName: action.name ?? null,
      };
    case "toggleLimitRange":
      return {
        ...state,
        selectedLimitRangeNamespace: action.namespace ?? null,
        selectedLimitRangeName: action.name ?? null,
      };
    case "toggleEndpoint":
      return {
        ...state,
        selectedEndpointNamespace: action.namespace ?? null,
        selectedEndpointName: action.name ?? null,
      };
    case "toggleEndpointSlice":
      return {
        ...state,
        selectedEndpointSliceNamespace: action.namespace ?? null,
        selectedEndpointSliceName: action.name ?? null,
      };
    case "toggleLease":
      return {
        ...state,
        selectedLeaseNamespace: action.namespace ?? null,
        selectedLeaseName: action.name ?? null,
      };
    case "togglePriorityClass":
      return { ...state, selectedPriorityClassName: action.name ?? null };
    case "togglePersistentVolume":
      return { ...state, selectedPersistentVolumeName: action.name ?? null };
    case "toggleStorageClass":
      return { ...state, selectedStorageClassName: action.name ?? null };
  }
}

const DetailDrawerCtx = createContext<DetailDrawerContextValue | null>(null);

export const useDetailDrawerContext = (): DetailDrawerContextValue => {
  const ctx = use(DetailDrawerCtx);
  if (!ctx) throw new Error("useDetailDrawerContext must be used inside DetailDrawerProvider");
  return ctx;
};

interface DetailDrawerProviderProps {
  children: ReactNode;
}

export const DetailDrawerProvider: FC<DetailDrawerProviderProps> = ({ children }) => {
  const [state, dispatch] = useReducer(detailDrawerReducer, initialState);

  const ctxValue = useMemo<DetailDrawerContextValue>(
    () => ({
      selectedNamespaceName: state.selectedNamespaceName,
      onToggleNamespaceDetail: (name) => dispatch({ type: "toggleNamespace", name }),

      selectedClusterRoleName: state.selectedClusterRoleName,
      onToggleClusterRoleDetail: (name) => dispatch({ type: "toggleClusterRole", name }),

      selectedClusterRoleBindingName: state.selectedClusterRoleBindingName,
      onToggleClusterRoleBindingDetail: (name) =>
        dispatch({ type: "toggleClusterRoleBinding", name }),

      selectedIngressClassName: state.selectedIngressClassName,
      onToggleIngressClassDetail: (name) => dispatch({ type: "toggleIngressClass", name }),

      selectedValidatingWebhookConfigName: state.selectedValidatingWebhookConfigName,
      onToggleValidatingWebhookConfigDetail: (name) =>
        dispatch({ type: "toggleValidatingWebhookConfig", name }),

      selectedRoleName: state.selectedRoleName,
      selectedRoleNamespace: state.selectedRoleNamespace,
      onToggleRoleDetail: (namespace, name) => dispatch({ type: "toggleRole", namespace, name }),

      selectedRoleBindingName: state.selectedRoleBindingName,
      selectedRoleBindingNamespace: state.selectedRoleBindingNamespace,
      onToggleRoleBindingDetail: (namespace, name) =>
        dispatch({ type: "toggleRoleBinding", namespace, name }),

      selectedServiceAccountName: state.selectedServiceAccountName,
      selectedServiceAccountNamespace: state.selectedServiceAccountNamespace,
      onToggleServiceAccountDetail: (namespace, name) =>
        dispatch({ type: "toggleServiceAccount", namespace, name }),

      selectedPodName: state.selectedPodName,
      selectedPodNamespace: state.selectedPodNamespace,
      onTogglePodDetail: (namespace, name) => dispatch({ type: "togglePod", namespace, name }),

      selectedJobName: state.selectedJobName,
      selectedJobNamespace: state.selectedJobNamespace,
      onToggleJobDetail: (namespace, name) => dispatch({ type: "toggleJob", namespace, name }),

      selectedCronJobName: state.selectedCronJobName,
      selectedCronJobNamespace: state.selectedCronJobNamespace,
      onToggleCronJobDetail: (namespace, name) =>
        dispatch({ type: "toggleCronJob", namespace, name }),

      selectedNodeName: state.selectedNodeName,
      onToggleNodeDetail: (name) => dispatch({ type: "toggleNode", name }),

      selectedServiceName: state.selectedServiceName,
      selectedServiceNamespace: state.selectedServiceNamespace,
      onToggleServiceDetail: (namespace, name) =>
        dispatch({ type: "toggleService", namespace, name }),

      selectedDeploymentName: state.selectedDeploymentName,
      selectedDeploymentNamespace: state.selectedDeploymentNamespace,
      onToggleDeploymentDetail: (namespace, name) =>
        dispatch({ type: "toggleDeployment", namespace, name }),

      selectedReplicaSetName: state.selectedReplicaSetName,
      selectedReplicaSetNamespace: state.selectedReplicaSetNamespace,
      onToggleReplicaSetDetail: (namespace, name) =>
        dispatch({ type: "toggleReplicaSet", namespace, name }),

      selectedDaemonSetName: state.selectedDaemonSetName,
      selectedDaemonSetNamespace: state.selectedDaemonSetNamespace,
      onToggleDaemonSetDetail: (namespace, name) =>
        dispatch({ type: "toggleDaemonSet", namespace, name }),

      selectedStatefulSetName: state.selectedStatefulSetName,
      selectedStatefulSetNamespace: state.selectedStatefulSetNamespace,
      onToggleStatefulSetDetail: (namespace, name) =>
        dispatch({ type: "toggleStatefulSet", namespace, name }),

      selectedEventName: state.selectedEventName,
      selectedEventNamespace: state.selectedEventNamespace,
      onToggleEventDetail: (namespace, name) => dispatch({ type: "toggleEvent", namespace, name }),

      selectedConfigMapName: state.selectedConfigMapName,
      selectedConfigMapNamespace: state.selectedConfigMapNamespace,
      onToggleConfigMapDetail: (namespace, name) =>
        dispatch({ type: "toggleConfigMap", namespace, name }),

      selectedNetworkPolicyName: state.selectedNetworkPolicyName,
      selectedNetworkPolicyNamespace: state.selectedNetworkPolicyNamespace,
      onToggleNetworkPolicyDetail: (namespace, name) =>
        dispatch({ type: "toggleNetworkPolicy", namespace, name }),

      selectedPersistentVolumeClaimName: state.selectedPersistentVolumeClaimName,
      selectedPersistentVolumeClaimNamespace: state.selectedPersistentVolumeClaimNamespace,
      onTogglePersistentVolumeClaimDetail: (namespace, name) =>
        dispatch({ type: "togglePersistentVolumeClaim", namespace, name }),

      selectedPodDisruptionBudgetName: state.selectedPodDisruptionBudgetName,
      selectedPodDisruptionBudgetNamespace: state.selectedPodDisruptionBudgetNamespace,
      onTogglePodDisruptionBudgetDetail: (namespace, name) =>
        dispatch({ type: "togglePodDisruptionBudget", namespace, name }),

      selectedSecretName: state.selectedSecretName,
      selectedSecretNamespace: state.selectedSecretNamespace,
      onToggleSecretDetail: (namespace, name) =>
        dispatch({ type: "toggleSecret", namespace, name }),

      selectedHPAName: state.selectedHPAName,
      selectedHPANamespace: state.selectedHPANamespace,
      onToggleHPADetail: (namespace, name) => dispatch({ type: "toggleHPA", namespace, name }),

      selectedIngressName: state.selectedIngressName,
      selectedIngressNamespace: state.selectedIngressNamespace,
      onToggleIngressDetail: (namespace, name) =>
        dispatch({ type: "toggleIngress", namespace, name }),

      selectedResourceQuotaName: state.selectedResourceQuotaName,
      selectedResourceQuotaNamespace: state.selectedResourceQuotaNamespace,
      onToggleResourceQuotaDetail: (namespace, name) =>
        dispatch({ type: "toggleResourceQuota", namespace, name }),

      selectedLimitRangeName: state.selectedLimitRangeName,
      selectedLimitRangeNamespace: state.selectedLimitRangeNamespace,
      onToggleLimitRangeDetail: (namespace, name) =>
        dispatch({ type: "toggleLimitRange", namespace, name }),

      selectedEndpointName: state.selectedEndpointName,
      selectedEndpointNamespace: state.selectedEndpointNamespace,
      onToggleEndpointDetail: (namespace, name) =>
        dispatch({ type: "toggleEndpoint", namespace, name }),

      selectedEndpointSliceName: state.selectedEndpointSliceName,
      selectedEndpointSliceNamespace: state.selectedEndpointSliceNamespace,
      onToggleEndpointSliceDetail: (namespace, name) =>
        dispatch({ type: "toggleEndpointSlice", namespace, name }),

      selectedLeaseName: state.selectedLeaseName,
      selectedLeaseNamespace: state.selectedLeaseNamespace,
      onToggleLease: (namespace, name) => dispatch({ type: "toggleLease", namespace, name }),

      selectedPriorityClassName: state.selectedPriorityClassName,
      onTogglePriorityClass: (name) => dispatch({ type: "togglePriorityClass", name }),

      selectedPersistentVolumeName: state.selectedPersistentVolumeName,
      onTogglePersistentVolumeDetail: (name) => dispatch({ type: "togglePersistentVolume", name }),

      selectedStorageClassName: state.selectedStorageClassName,
      onToggleStorageClassDetail: (name) => dispatch({ type: "toggleStorageClass", name }),
    }),
    [state]
  );

  return <DetailDrawerCtx.Provider value={ctxValue}>{children}</DetailDrawerCtx.Provider>;
};

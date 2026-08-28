package kubeResources

import (
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/litelensapp/litelens/packages/core/kube/dto"
	appsv1 "k8s.io/api/apps/v1"
	"k8s.io/apimachinery/pkg/labels"
	listersappsv1 "k8s.io/client-go/listers/apps/v1"
)

func toDaemonSet(ds *appsv1.DaemonSet) dto.DaemonSet {
	pods := fmt.Sprintf("%d/%d", ds.Status.NumberReady, ds.Status.DesiredNumberScheduled)

	selectorParts := make([]string, 0, len(ds.Spec.Template.Spec.NodeSelector))
	for k, v := range ds.Spec.Template.Spec.NodeSelector {
		selectorParts = append(selectorParts, fmt.Sprintf("%s=%s", k, v))
	}
	sort.Strings(selectorParts)
	nodeSelector := strings.Join(selectorParts, ", ")
	if nodeSelector == "" {
		nodeSelector = "<none>"
	}

	return dto.DaemonSet{
		Name:         ds.Name,
		Namespace:    ds.Namespace,
		Pods:         pods,
		NodeSelector: nodeSelector,
		Age:          humanAge(ds.CreationTimestamp.Time),
		CreatedAt:    ds.CreationTimestamp.Format(time.RFC3339),
		Labels: func() map[string]string {
			if ds.Labels == nil {
				return map[string]string{}
			}
			return ds.Labels
		}(),
		Annotations: func() map[string]string {
			if ds.Annotations == nil {
				return map[string]string{}
			}
			return ds.Annotations
		}(),
		ManagedFields: toManagedFields(ds),
		Selector: func() string {
			if ds.Spec.Selector == nil {
				return ""
			}
			keys := make([]string, 0, len(ds.Spec.Selector.MatchLabels))
			for k := range ds.Spec.Selector.MatchLabels {
				keys = append(keys, k)
			}
			sort.Strings(keys)
			parts := make([]string, 0, len(keys))
			for _, k := range keys {
				parts = append(parts, k+"="+ds.Spec.Selector.MatchLabels[k])
			}
			return strings.Join(parts, ", ")
		}(),
		Images: func() []string {
			out := make([]string, 0, len(ds.Spec.Template.Spec.Containers))
			for _, c := range ds.Spec.Template.Spec.Containers {
				out = append(out, c.Image)
			}
			return out
		}(),
		StrategyType: string(ds.Spec.UpdateStrategy.Type),
		Tolerations:  len(ds.Spec.Template.Spec.Tolerations),
		PodStatus: fmt.Sprintf("%d desired, %d ready, %d available, %d unavailable",
			ds.Status.DesiredNumberScheduled, ds.Status.NumberReady,
			ds.Status.NumberAvailable, ds.Status.NumberUnavailable),
	}
}

func GetDaemonSetByName(lister listersappsv1.DaemonSetLister, namespace, name string) (dto.DaemonSet, error) {
	ds, err := lister.DaemonSets(namespace).Get(name)
	if err != nil {
		return dto.DaemonSet{}, err
	}
	return toDaemonSet(ds), nil
}

func ListDaemonSets(lister listersappsv1.DaemonSetLister, namespaces []string) ([]dto.DaemonSet, error) {
	dss, err := lister.List(labels.Everything())
	if err != nil {
		return nil, err
	}
	dss = filterByNamespaces(dss, namespaces)
	result := make([]dto.DaemonSet, len(dss))
	for i, ds := range dss {
		result[i] = toDaemonSet(ds)
	}
	return result, nil
}

func SummarizeDaemonSets(dss []*appsv1.DaemonSet) dto.DaemonSetSummary {
	summary := dto.DaemonSetSummary{}
	for _, ds := range dss {
		desired := ds.Status.DesiredNumberScheduled
		ready := ds.Status.NumberReady
		if desired > 0 && ready >= desired {
			summary.Running++
		} else {
			summary.Pending++
		}
	}
	return summary
}

package kubeResources

import (
	"fmt"
	"log"
	"sort"
	"strings"
	"time"

	"github.com/litelensapp/litelens/packages/core/kube/dto"
	appsv1 "k8s.io/api/apps/v1"
	"k8s.io/apimachinery/pkg/labels"
	listersappsv1 "k8s.io/client-go/listers/apps/v1"
)

func toStatefulSet(ss *appsv1.StatefulSet) dto.StatefulSet {
	var replicas int32 = 1
	if ss.Spec.Replicas != nil {
		replicas = *ss.Spec.Replicas
	}
	return dto.StatefulSet{
		Name:      ss.Name,
		Namespace: ss.Namespace,
		Pods:      fmt.Sprintf("%d/%d", ss.Status.ReadyReplicas, replicas),
		Replicas:  replicas,
		Age:       humanAge(ss.CreationTimestamp.Time),
		CreatedAt: ss.CreationTimestamp.Format(time.RFC3339),
		Labels: func() map[string]string {
			if ss.Labels == nil {
				return map[string]string{}
			}
			return ss.Labels
		}(),
		Annotations: func() map[string]string {
			if ss.Annotations == nil {
				return map[string]string{}
			}
			return ss.Annotations
		}(),
		ManagedFields: toManagedFields(ss),
		Selector: func() string {
			if ss.Spec.Selector == nil {
				return ""
			}
			keys := make([]string, 0, len(ss.Spec.Selector.MatchLabels))
			for k := range ss.Spec.Selector.MatchLabels {
				keys = append(keys, k)
			}
			sort.Strings(keys)
			parts := make([]string, 0, len(keys))
			for _, k := range keys {
				parts = append(parts, k+"="+ss.Spec.Selector.MatchLabels[k])
			}
			return strings.Join(parts, ", ")
		}(),
		Images: func() []string {
			out := make([]string, 0, len(ss.Spec.Template.Spec.Containers))
			for _, c := range ss.Spec.Template.Spec.Containers {
				out = append(out, c.Image)
			}
			return out
		}(),
		Affinities: func() int {
			if ss.Spec.Template.Spec.Affinity != nil {
				return 1
			}
			return 0
		}(),
		PodStatus: fmt.Sprintf("%d desired, %d ready, %d available", replicas, ss.Status.ReadyReplicas, ss.Status.AvailableReplicas),
	}
}

func ListStatefulSets(lister listersappsv1.StatefulSetLister, namespaces []string) ([]dto.StatefulSet, error) {
	var sss []*appsv1.StatefulSet
	if len(namespaces) == 0 {
		all, err := lister.List(labels.Everything())
		if err != nil {
			return nil, err
		}
		sss = all
	} else {
		for _, ns := range namespaces {
			nsStatefulSets, err := lister.StatefulSets(ns).List(labels.Everything())
			if err != nil {
				// Tolerate per-namespace errors (e.g., RBAC 403) but log them so
				// genuine failures (API server errors, etc.) remain visible.
				log.Printf("kubeResources: ListStatefulSets: namespace %q: %v", ns, err)
				continue
			}
			sss = append(sss, nsStatefulSets...)
		}
	}
	result := make([]dto.StatefulSet, len(sss))
	for i, ss := range sss {
		result[i] = toStatefulSet(ss)
	}
	return result, nil
}

func SummarizeStatefulSets(sss []*appsv1.StatefulSet) dto.StatefulSetSummary {
	summary := dto.StatefulSetSummary{}
	for _, ss := range sss {
		desired := ss.Status.Replicas
		ready := ss.Status.ReadyReplicas
		if desired > 0 && ready >= desired {
			summary.Running++
		} else {
			summary.Pending++
		}
	}
	return summary
}

func GetStatefulSetByName(lister listersappsv1.StatefulSetLister, namespace, name string) (dto.StatefulSet, error) {
	ss, err := lister.StatefulSets(namespace).Get(name)
	if err != nil {
		return dto.StatefulSet{}, err
	}
	return toStatefulSet(ss), nil
}

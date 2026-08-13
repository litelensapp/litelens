package kubeResources

import (
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/litelensapp/litelens/internal/dto"
	appsv1 "k8s.io/api/apps/v1"
	"k8s.io/apimachinery/pkg/labels"
	listersappsv1 "k8s.io/client-go/listers/apps/v1"
	sigsyaml "sigs.k8s.io/yaml"
)

func toReplicaSet(rs *appsv1.ReplicaSet) dto.ReplicaSet {
	var desired int32
	if rs.Spec.Replicas != nil {
		desired = *rs.Spec.Replicas
	}
	return dto.ReplicaSet{
		Name:      rs.Name,
		Namespace: rs.Namespace,
		Desired:   desired,
		Current:   rs.Status.Replicas,
		Ready:     rs.Status.ReadyReplicas,
		Age:       humanAge(rs.CreationTimestamp.Time),
		OwnerName: func() string {
			if len(rs.OwnerReferences) > 0 {
				return rs.OwnerReferences[0].Name
			}
			return ""
		}(),
		CreatedAt: rs.CreationTimestamp.Format(time.RFC3339),
		OwnerKind: func() string {
			if len(rs.OwnerReferences) > 0 {
				return rs.OwnerReferences[0].Kind
			}
			return ""
		}(),
		Labels: func() map[string]string {
			if rs.Labels == nil {
				return map[string]string{}
			}
			return rs.Labels
		}(),
		Annotations: func() map[string]string {
			if rs.Annotations == nil {
				return map[string]string{}
			}
			return rs.Annotations
		}(),
		ManagedFields: func() []dto.ManagedField {
			out := make([]dto.ManagedField, 0, len(rs.ManagedFields))
			for _, mf := range rs.ManagedFields {
				fieldsYAML := ""
				if raw := mf.FieldsV1.GetRawBytes(); len(raw) > 0 {
					if yamlBytes, err := sigsyaml.JSONToYAML(raw); err == nil {
						fieldsYAML = string(yamlBytes)
					}
				}
				out = append(out, dto.ManagedField{
					Manager:    mf.Manager,
					Operation:  string(mf.Operation),
					FieldsYAML: fieldsYAML,
				})
			}
			return out
		}(),
		Selector: func() string {
			if rs.Spec.Selector == nil {
				return ""
			}
			keys := make([]string, 0, len(rs.Spec.Selector.MatchLabels))
			for k := range rs.Spec.Selector.MatchLabels {
				keys = append(keys, k)
			}
			sort.Strings(keys)
			parts := make([]string, 0, len(keys))
			for _, k := range keys {
				parts = append(parts, k+"="+rs.Spec.Selector.MatchLabels[k])
			}
			return strings.Join(parts, ", ")
		}(),
		NodeSelector: func() string {
			ns := rs.Spec.Template.Spec.NodeSelector
			if len(ns) == 0 {
				return ""
			}
			keys := make([]string, 0, len(ns))
			for k := range ns {
				keys = append(keys, k)
			}
			sort.Strings(keys)
			parts := make([]string, 0, len(keys))
			for _, k := range keys {
				parts = append(parts, k+"="+ns[k])
			}
			return strings.Join(parts, ", ")
		}(),
		Images: func() []string {
			out := make([]string, 0, len(rs.Spec.Template.Spec.Containers))
			for _, c := range rs.Spec.Template.Spec.Containers {
				out = append(out, c.Image)
			}
			return out
		}(),
		ReplicasDetail: fmt.Sprintf("%d current / %d desired", rs.Status.Replicas, desired),
		Tolerations:    len(rs.Spec.Template.Spec.Tolerations),
		Affinities: func() int {
			if rs.Spec.Template.Spec.Affinity != nil {
				return 1
			}
			return 0
		}(),
		PodStatus: fmt.Sprintf("%d desired, %d ready, %d available",
			desired, rs.Status.ReadyReplicas, rs.Status.AvailableReplicas),
	}
}

func GetReplicaSetByName(lister listersappsv1.ReplicaSetLister, namespace, name string) (dto.ReplicaSet, error) {
	rs, err := lister.ReplicaSets(namespace).Get(name)
	if err != nil {
		return dto.ReplicaSet{}, err
	}
	return toReplicaSet(rs), nil
}

func ListReplicaSets(lister listersappsv1.ReplicaSetLister, namespace string) ([]dto.ReplicaSet, error) {
	var rss []*appsv1.ReplicaSet
	var err error
	if namespace == "" {
		rss, err = lister.List(labels.Everything())
	} else {
		rss, err = lister.ReplicaSets(namespace).List(labels.Everything())
	}
	if err != nil {
		return nil, err
	}
	result := make([]dto.ReplicaSet, len(rss))
	for i, rs := range rss {
		result[i] = toReplicaSet(rs)
	}
	return result, nil
}

func SummarizeReplicaSets(rss []*appsv1.ReplicaSet) dto.ReplicaSetSummary {
	summary := dto.ReplicaSetSummary{}
	for _, rs := range rss {
		if rs.Status.ReadyReplicas >= rs.Status.Replicas {
			summary.Running++
		} else {
			summary.Pending++
		}
	}
	return summary
}

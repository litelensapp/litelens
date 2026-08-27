package kubeResources

import (
	"time"

	"github.com/litelensapp/litelens/packages/core/kube/dto"
	"k8s.io/apimachinery/pkg/labels"
	listerscorev1 "k8s.io/client-go/listers/core/v1"
	sigsyaml "sigs.k8s.io/yaml"
)

func ListNamespaces(lister listerscorev1.NamespaceLister) ([]dto.Namespace, error) {
	nss, err := lister.List(labels.Everything())
	if err != nil {
		return nil, err
	}
	result := make([]dto.Namespace, len(nss))
	for i, ns := range nss {
		managedFields := make([]dto.ManagedField, 0, len(ns.ManagedFields))
		for _, mf := range ns.ManagedFields {
			fieldsYAML := ""
			if mf.FieldsV1 != nil {
				if raw := mf.FieldsV1.GetRawBytes(); len(raw) > 0 {
					if yamlBytes, err := sigsyaml.JSONToYAML(raw); err == nil {
						fieldsYAML = string(yamlBytes)
					}
				}
			}
			managedFields = append(managedFields, dto.ManagedField{
				Manager:    mf.Manager,
				Operation:  string(mf.Operation),
				FieldsYAML: fieldsYAML,
			})
		}

		lbls := ns.Labels
		if lbls == nil {
			lbls = map[string]string{}
		}
		annotations := ns.Annotations
		if annotations == nil {
			annotations = map[string]string{}
		}

		result[i] = dto.Namespace{
			Name:           ns.Name,
			Labels:         lbls,
			Annotations:    annotations,
			Age:            humanAge(ns.CreationTimestamp.Time),
			CreatedAt:      ns.CreationTimestamp.UTC().Format(time.RFC3339),
			Status:         string(ns.Status.Phase),
			ManagedFields:  managedFields,
			ResourceQuotas: []string{},
			LimitRanges:    []string{},
		}
	}
	return result, nil
}

func GetNamespaceByName(lister listerscorev1.NamespaceLister, name string) (dto.Namespace, error) {
	ns, err := lister.Get(name)
	if err != nil {
		return dto.Namespace{}, err
	}
	managedFields := make([]dto.ManagedField, 0, len(ns.ManagedFields))
	for _, mf := range ns.ManagedFields {
		fieldsYAML := ""
		if mf.FieldsV1 != nil {
			if raw := mf.FieldsV1.GetRawBytes(); len(raw) > 0 {
				if yamlBytes, err := sigsyaml.JSONToYAML(raw); err == nil {
					fieldsYAML = string(yamlBytes)
				}
			}
		}
		managedFields = append(managedFields, dto.ManagedField{
			Manager:    mf.Manager,
			Operation:  string(mf.Operation),
			FieldsYAML: fieldsYAML,
		})
	}
	lbls := ns.Labels
	if lbls == nil {
		lbls = map[string]string{}
	}
	annotations := ns.Annotations
	if annotations == nil {
		annotations = map[string]string{}
	}
	return dto.Namespace{
		Name:           ns.Name,
		Labels:         lbls,
		Annotations:    annotations,
		Age:            humanAge(ns.CreationTimestamp.Time),
		CreatedAt:      ns.CreationTimestamp.UTC().Format(time.RFC3339),
		Status:         string(ns.Status.Phase),
		ManagedFields:  managedFields,
		ResourceQuotas: []string{},
		LimitRanges:    []string{},
	}, nil
}

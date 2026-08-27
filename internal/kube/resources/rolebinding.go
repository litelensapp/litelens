package kubeResources

import (
	"strings"
	"time"

	"github.com/litelensapp/litelens/packages/core/kube/dto"
	rbacv1 "k8s.io/api/rbac/v1"
	"k8s.io/apimachinery/pkg/labels"
	listersrbacv1 "k8s.io/client-go/listers/rbac/v1"
	sigsyaml "sigs.k8s.io/yaml"
)

func toRoleBinding(rb *rbacv1.RoleBinding) dto.RoleBinding {
	names := make([]string, 0, len(rb.Subjects))
	seen := make(map[string]struct{})
	kinds := make([]string, 0)
	subjects := make([]dto.RBSubject, 0, len(rb.Subjects))
	for _, s := range rb.Subjects {
		names = append(names, s.Name)
		subjects = append(subjects, dto.RBSubject{Kind: s.Kind, Name: s.Name, Namespace: s.Namespace})
		if _, ok := seen[s.Kind]; !ok {
			seen[s.Kind] = struct{}{}
			kinds = append(kinds, s.Kind)
		}
	}
	bindings := strings.Join(names, ", ")
	if bindings == "" {
		bindings = "-"
	}

	managedFields := make([]dto.ManagedField, 0, len(rb.ManagedFields))
	for _, mf := range rb.ManagedFields {
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

	lbls := rb.Labels
	if lbls == nil {
		lbls = map[string]string{}
	}
	annotations := rb.Annotations
	if annotations == nil {
		annotations = map[string]string{}
	}

	return dto.RoleBinding{
		Name:          rb.Name,
		Namespace:     rb.Namespace,
		Bindings:      bindings,
		Age:           humanAge(rb.CreationTimestamp.Time),
		RoleRefName:   rb.RoleRef.Name,
		Types:         strings.Join(kinds, ", "),
		CreatedAt:     rb.CreationTimestamp.UTC().Format(time.RFC3339),
		Labels:        lbls,
		Annotations:   annotations,
		ManagedFields: managedFields,
		RoleRefKind:   rb.RoleRef.Kind,
		RoleRefGroup:  rb.RoleRef.APIGroup,
		Subjects:      subjects,
	}
}

func GetRoleBindingByName(lister listersrbacv1.RoleBindingLister, namespace, name string) (dto.RoleBinding, error) {
	rb, err := lister.RoleBindings(namespace).Get(name)
	if err != nil {
		return dto.RoleBinding{}, err
	}
	return toRoleBinding(rb), nil
}

func ListRoleBindings(lister listersrbacv1.RoleBindingLister, namespaces []string) ([]dto.RoleBinding, error) {
	rbs, err := lister.List(labels.Everything())
	if err != nil {
		return nil, err
	}
	if len(namespaces) > 0 {
		nsSet := make(map[string]struct{}, len(namespaces))
		for _, ns := range namespaces {
			nsSet[ns] = struct{}{}
		}
		filtered := rbs[:0:0]
		for _, rb := range rbs {
			if _, ok := nsSet[rb.Namespace]; ok {
				filtered = append(filtered, rb)
			}
		}
		rbs = filtered
	}
	result := make([]dto.RoleBinding, len(rbs))
	for i, rb := range rbs {
		result[i] = toRoleBinding(rb)
	}
	return result, nil
}

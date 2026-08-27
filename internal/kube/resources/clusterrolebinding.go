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

func toClusterRoleBinding(crb *rbacv1.ClusterRoleBinding) dto.ClusterRoleBinding {
	names := make([]string, 0, len(crb.Subjects))
	subjects := make([]dto.CRBSubject, 0, len(crb.Subjects))
	for _, s := range crb.Subjects {
		names = append(names, s.Name)
		subjects = append(subjects, dto.CRBSubject{
			Kind:      s.Kind,
			Name:      s.Name,
			Namespace: s.Namespace,
		})
	}
	bindings := strings.Join(names, ", ")
	if bindings == "" {
		bindings = "-"
	}

	managedFields := make([]dto.ManagedField, 0, len(crb.ManagedFields))
	for _, mf := range crb.ManagedFields {
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

	lbls := crb.Labels
	if lbls == nil {
		lbls = map[string]string{}
	}
	annotations := crb.Annotations
	if annotations == nil {
		annotations = map[string]string{}
	}

	return dto.ClusterRoleBinding{
		Name:          crb.Name,
		Bindings:      bindings,
		Age:           humanAge(crb.CreationTimestamp.Time),
		CreatedAt:     crb.CreationTimestamp.UTC().Format(time.RFC3339),
		Labels:        lbls,
		Annotations:   annotations,
		ManagedFields: managedFields,
		RoleRefKind:   crb.RoleRef.Kind,
		RoleRefName:   crb.RoleRef.Name,
		RoleRefGroup:  crb.RoleRef.APIGroup,
		Subjects:      subjects,
	}
}

func GetClusterRoleBindingByName(lister listersrbacv1.ClusterRoleBindingLister, name string) (dto.ClusterRoleBinding, error) {
	crb, err := lister.Get(name)
	if err != nil {
		return dto.ClusterRoleBinding{}, err
	}
	return toClusterRoleBinding(crb), nil
}

func ListClusterRoleBindings(lister listersrbacv1.ClusterRoleBindingLister) ([]dto.ClusterRoleBinding, error) {
	crbs, err := lister.List(labels.Everything())
	if err != nil {
		return nil, err
	}
	result := make([]dto.ClusterRoleBinding, len(crbs))
	for i, crb := range crbs {
		result[i] = toClusterRoleBinding(crb)
	}
	return result, nil
}

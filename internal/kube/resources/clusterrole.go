package kubeResources

import (
	"time"

	"github.com/litelensapp/litelens/internal/dto"
	rbacv1 "k8s.io/api/rbac/v1"
	"k8s.io/apimachinery/pkg/labels"
	listersrbacv1 "k8s.io/client-go/listers/rbac/v1"
	sigsyaml "sigs.k8s.io/yaml"
)

func toClusterRole(cr *rbacv1.ClusterRole) dto.ClusterRole {
	managedFields := make([]dto.ManagedField, 0, len(cr.ManagedFields))
	for _, mf := range cr.ManagedFields {
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

	lbls := cr.Labels
	if lbls == nil {
		lbls = map[string]string{}
	}
	annotations := cr.Annotations
	if annotations == nil {
		annotations = map[string]string{}
	}

	rules := make([]dto.PolicyRule, 0, len(cr.Rules))
	for _, r := range cr.Rules {
		resources := r.Resources
		if resources == nil {
			resources = []string{}
		}
		verbs := r.Verbs
		if verbs == nil {
			verbs = []string{}
		}
		apiGroups := r.APIGroups
		if apiGroups == nil {
			apiGroups = []string{}
		}
		resourceNames := r.ResourceNames
		if resourceNames == nil {
			resourceNames = []string{}
		}
		nonResourceURLs := r.NonResourceURLs
		if nonResourceURLs == nil {
			nonResourceURLs = []string{}
		}
		rules = append(rules, dto.PolicyRule{
			Resources:       resources,
			Verbs:           verbs,
			APIGroups:       apiGroups,
			ResourceNames:   resourceNames,
			NonResourceURLs: nonResourceURLs,
		})
	}

	return dto.ClusterRole{
		Name:          cr.Name,
		Age:           humanAge(cr.CreationTimestamp.Time),
		CreatedAt:     cr.CreationTimestamp.UTC().Format(time.RFC3339),
		Labels:        lbls,
		Annotations:   annotations,
		ManagedFields: managedFields,
		Rules:         rules,
	}
}

func GetClusterRoleByName(lister listersrbacv1.ClusterRoleLister, name string) (dto.ClusterRole, error) {
	cr, err := lister.Get(name)
	if err != nil {
		return dto.ClusterRole{}, err
	}
	return toClusterRole(cr), nil
}

func ListClusterRoles(lister listersrbacv1.ClusterRoleLister) ([]dto.ClusterRole, error) {
	crs, err := lister.List(labels.Everything())
	if err != nil {
		return nil, err
	}
	result := make([]dto.ClusterRole, len(crs))
	for i, cr := range crs {
		result[i] = toClusterRole(cr)
	}
	return result, nil
}

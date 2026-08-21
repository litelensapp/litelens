package kubeResources

import (
	"time"

	"github.com/litelensapp/litelens/packages/core/dto"
	rbacv1 "k8s.io/api/rbac/v1"
	"k8s.io/apimachinery/pkg/labels"
	listersrbacv1 "k8s.io/client-go/listers/rbac/v1"
	sigsyaml "sigs.k8s.io/yaml"
)

func toRole(r *rbacv1.Role) dto.Role {
	managedFields := make([]dto.ManagedField, 0, len(r.ManagedFields))
	for _, mf := range r.ManagedFields {
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

	lbls := r.Labels
	if lbls == nil {
		lbls = map[string]string{}
	}
	annotations := r.Annotations
	if annotations == nil {
		annotations = map[string]string{}
	}

	rules := make([]dto.PolicyRule, 0, len(r.Rules))
	for _, rule := range r.Rules {
		resources := rule.Resources
		if resources == nil {
			resources = []string{}
		}
		verbs := rule.Verbs
		if verbs == nil {
			verbs = []string{}
		}
		apiGroups := rule.APIGroups
		if apiGroups == nil {
			apiGroups = []string{}
		}
		resourceNames := rule.ResourceNames
		if resourceNames == nil {
			resourceNames = []string{}
		}
		nonResourceURLs := rule.NonResourceURLs
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

	return dto.Role{
		Name:          r.Name,
		Namespace:     r.Namespace,
		Age:           humanAge(r.CreationTimestamp.Time),
		CreatedAt:     r.CreationTimestamp.UTC().Format(time.RFC3339),
		Labels:        lbls,
		Annotations:   annotations,
		ManagedFields: managedFields,
		Rules:         rules,
	}
}

func GetRoleByName(lister listersrbacv1.RoleLister, namespace, name string) (dto.Role, error) {
	r, err := lister.Roles(namespace).Get(name)
	if err != nil {
		return dto.Role{}, err
	}
	return toRole(r), nil
}

func ListRoles(lister listersrbacv1.RoleLister, namespaces []string) ([]dto.Role, error) {
	roles, err := lister.List(labels.Everything())
	if err != nil {
		return nil, err
	}
	if len(namespaces) > 0 {
		nsSet := make(map[string]struct{}, len(namespaces))
		for _, ns := range namespaces {
			nsSet[ns] = struct{}{}
		}
		filtered := roles[:0:0]
		for _, role := range roles {
			if _, ok := nsSet[role.Namespace]; ok {
				filtered = append(filtered, role)
			}
		}
		roles = filtered
	}
	result := make([]dto.Role, len(roles))
	for i, r := range roles {
		result[i] = toRole(r)
	}
	return result, nil
}

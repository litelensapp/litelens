package kubeResources

import (
	"context"
	"sort"
	"time"

	"github.com/litelensapp/litelens/packages/core/kube/dto"
	corev1 "k8s.io/api/core/v1"
	corev1client "k8s.io/client-go/kubernetes/typed/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	listerscorev1 "k8s.io/client-go/listers/core/v1"
	sigsyaml "sigs.k8s.io/yaml"
)

func toConfigMap(cm *corev1.ConfigMap) dto.ConfigMap {
	keys := make([]string, 0, len(cm.Data)+len(cm.BinaryData))
	for k := range cm.Data {
		keys = append(keys, k)
	}
	for k := range cm.BinaryData {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	managedFields := make([]dto.ManagedField, 0, len(cm.ManagedFields))
	for _, mf := range cm.ManagedFields {
		fieldsYAML := ""
		if raw := mf.FieldsV1.GetRawBytes(); len(raw) > 0 {
			if yamlBytes, err := sigsyaml.JSONToYAML(raw); err == nil {
				fieldsYAML = string(yamlBytes)
			}
		}
		managedFields = append(managedFields, dto.ManagedField{
			Manager:    mf.Manager,
			Operation:  string(mf.Operation),
			FieldsYAML: fieldsYAML,
		})
	}

	return dto.ConfigMap{
		Name:          cm.Name,
		Namespace:     cm.Namespace,
		Keys:          keys,
		Age:           humanAge(cm.CreationTimestamp.Time),
		CreatedAt:     cm.CreationTimestamp.UTC().Format(time.RFC3339),
		Labels:        cm.Labels,
		Annotations:   cm.Annotations,
		ManagedFields: managedFields,
		Data:          cm.Data,
	}
}

func ListConfigMaps(lister listerscorev1.ConfigMapLister, namespaces []string) ([]dto.ConfigMap, error) {
	cms, err := lister.List(labels.Everything())
	if err != nil {
		return nil, err
	}
	if len(namespaces) > 0 {
		nsSet := make(map[string]struct{}, len(namespaces))
		for _, ns := range namespaces {
			nsSet[ns] = struct{}{}
		}
		filtered := cms[:0:0]
		for _, cm := range cms {
			if _, ok := nsSet[cm.Namespace]; ok {
				filtered = append(filtered, cm)
			}
		}
		cms = filtered
	}
	result := make([]dto.ConfigMap, len(cms))
	for i, cm := range cms {
		result[i] = toConfigMap(cm)
	}
	return result, nil
}

func GetConfigMapByName(lister listerscorev1.ConfigMapLister, namespace, name string) (dto.ConfigMap, error) {
	cm, err := lister.ConfigMaps(namespace).Get(name)
	if err != nil {
		return dto.ConfigMap{}, err
	}
	return toConfigMap(cm), nil
}

func UpdateConfigMap(client corev1client.ConfigMapInterface, cm *corev1.ConfigMap, data map[string]string) (*corev1.ConfigMap, error) {
	updated := cm.DeepCopy()
	updated.Data = data
	return client.Update(context.Background(), updated, metav1.UpdateOptions{})
}

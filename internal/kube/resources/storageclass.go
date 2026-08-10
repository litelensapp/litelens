package kubeResources

import (
	"time"

	"github.com/litelensapp/litelens/internal/dto"
	storagev1 "k8s.io/api/storage/v1"
	"k8s.io/apimachinery/pkg/labels"
	listersstoragev1 "k8s.io/client-go/listers/storage/v1"
	sigsyaml "sigs.k8s.io/yaml"
)

func toStorageClass(sc *storagev1.StorageClass) dto.StorageClass {
	reclaimPolicy := ""
	if sc.ReclaimPolicy != nil {
		reclaimPolicy = string(*sc.ReclaimPolicy)
	}

	isDefault := sc.Annotations["storageclass.kubernetes.io/is-default-class"] == "true"

	volumeBindingMode := ""
	if sc.VolumeBindingMode != nil {
		volumeBindingMode = string(*sc.VolumeBindingMode)
	}

	managedFields := make([]dto.ManagedField, 0, len(sc.ManagedFields))
	for _, mf := range sc.ManagedFields {
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

	labels := sc.Labels
	if labels == nil {
		labels = map[string]string{}
	}
	annotations := sc.Annotations
	if annotations == nil {
		annotations = map[string]string{}
	}
	parameters := sc.Parameters
	if parameters == nil {
		parameters = map[string]string{}
	}
	mountOptions := sc.MountOptions
	if mountOptions == nil {
		mountOptions = []string{}
	}

	return dto.StorageClass{
		Name:              sc.Name,
		Provisioner:       sc.Provisioner,
		ReclaimPolicy:     reclaimPolicy,
		Default:           isDefault,
		Age:               humanAge(sc.CreationTimestamp.Time),
		CreatedAt:         sc.CreationTimestamp.UTC().Format(time.RFC3339),
		Labels:            labels,
		Annotations:       annotations,
		ManagedFields:     managedFields,
		VolumeBindingMode: volumeBindingMode,
		MountOptions:      mountOptions,
		Parameters:        parameters,
	}
}

func ListStorageClasses(scLister listersstoragev1.StorageClassLister) ([]dto.StorageClass, error) {
	scs, err := scLister.List(labels.Everything())
	if err != nil {
		return nil, err
	}
	result := make([]dto.StorageClass, len(scs))
	for i, sc := range scs {
		result[i] = toStorageClass(sc)
	}
	return result, nil
}

func GetStorageClassByName(lister listersstoragev1.StorageClassLister, name string) (dto.StorageClass, error) {
	sc, err := lister.Get(name)
	if err != nil {
		return dto.StorageClass{}, err
	}
	return toStorageClass(sc), nil
}

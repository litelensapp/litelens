package kubeResources

import (
	"sort"
	"time"

	"github.com/litelensapp/litelens/internal/dto"
	coordinationv1 "k8s.io/api/coordination/v1"
	"k8s.io/apimachinery/pkg/labels"
	listerscoordinationv1 "k8s.io/client-go/listers/coordination/v1"
	sigsyaml "sigs.k8s.io/yaml"
)

func toLease(l *coordinationv1.Lease) dto.Lease {
	holderIdentity := ""
	if l.Spec.HolderIdentity != nil {
		holderIdentity = *l.Spec.HolderIdentity
	}

	var leaseDurationSeconds int32
	if l.Spec.LeaseDurationSeconds != nil {
		leaseDurationSeconds = *l.Spec.LeaseDurationSeconds
	}

	var leaseTransitions int32
	if l.Spec.LeaseTransitions != nil {
		leaseTransitions = *l.Spec.LeaseTransitions
	}

	renewTime := ""
	if l.Spec.RenewTime != nil {
		renewTime = l.Spec.RenewTime.UTC().Format(time.RFC3339)
	}

	acquireTime := ""
	if l.Spec.AcquireTime != nil {
		acquireTime = l.Spec.AcquireTime.UTC().Format(time.RFC3339)
	}

	managedFields := make([]dto.ManagedField, 0, len(l.ManagedFields))
	for _, mf := range l.ManagedFields {
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

	return dto.Lease{
		Name:                 l.Name,
		Namespace:            l.Namespace,
		HolderIdentity:       holderIdentity,
		LeaseDurationSeconds: leaseDurationSeconds,
		RenewTime:            renewTime,
		AcquireTime:          acquireTime,
		LeaseTransitions:     leaseTransitions,
		Age:                  humanAge(l.CreationTimestamp.Time),
		CreatedAt:            l.CreationTimestamp.UTC().Format(time.RFC3339),
		Labels:               l.Labels,
		Annotations:          l.Annotations,
		ManagedFields:        managedFields,
	}
}

func ListLeases(lister listerscoordinationv1.LeaseLister, namespace string) ([]dto.Lease, error) {
	var leases []*coordinationv1.Lease
	var err error
	if namespace == "" {
		leases, err = lister.List(labels.Everything())
	} else {
		leases, err = lister.Leases(namespace).List(labels.Everything())
	}
	if err != nil {
		return nil, err
	}
	result := make([]dto.Lease, len(leases))
	for i, l := range leases {
		result[i] = toLease(l)
	}
	sort.Slice(result, func(i, j int) bool {
		return result[i].Name < result[j].Name
	})
	return result, nil
}

func GetLeaseByName(lister listerscoordinationv1.LeaseLister, namespace, name string) (dto.Lease, error) {
	lease, err := lister.Leases(namespace).Get(name)
	if err != nil {
		return dto.Lease{}, err
	}
	return toLease(lease), nil
}

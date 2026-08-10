package kubeResources

import (
	"fmt"
	"strings"
	"time"

	"github.com/litelensapp/litelens/internal/dto"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/labels"
	listerscorev1 "k8s.io/client-go/listers/core/v1"
	sigsyaml "sigs.k8s.io/yaml"
)

func toEndpoint(ep *corev1.Endpoints) dto.Endpoint { //nolint:staticcheck
	var addrs []string
	for _, subset := range ep.Subsets {
		for _, addr := range subset.Addresses {
			if len(subset.Ports) == 0 {
				addrs = append(addrs, addr.IP)
			}
			for _, port := range subset.Ports {
				addrs = append(addrs, fmt.Sprintf("%s:%d", addr.IP, port.Port))
			}
		}
	}
	endpoints := strings.Join(addrs, ", ")
	if endpoints == "" {
		endpoints = "<none>"
	}

	managedFields := make([]dto.ManagedField, 0, len(ep.ManagedFields))
	for _, mf := range ep.ManagedFields {
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

	subsets := make([]dto.EndpointSubset, 0, len(ep.Subsets))
	for _, subset := range ep.Subsets {
		dtoAddresses := make([]dto.EndpointAddress, 0, len(subset.Addresses))
		for _, a := range subset.Addresses {
			targetName := ""
			if a.TargetRef != nil {
				targetName = a.TargetRef.Name
			}
			dtoAddresses = append(dtoAddresses, dto.EndpointAddress{
				IP:         a.IP,
				Hostname:   a.Hostname,
				TargetName: targetName,
			})
		}
		dtoPorts := make([]dto.EndpointPort, 0, len(subset.Ports))
		for _, p := range subset.Ports {
			dtoPorts = append(dtoPorts, dto.EndpointPort{
				Name:     p.Name,
				Port:     p.Port,
				Protocol: string(p.Protocol),
			})
		}
		subsets = append(subsets, dto.EndpointSubset{
			Addresses: dtoAddresses,
			Ports:     dtoPorts,
		})
	}

	return dto.Endpoint{
		Name:          ep.Name,
		Namespace:     ep.Namespace,
		Endpoints:     endpoints,
		Age:           humanAge(ep.CreationTimestamp.Time),
		CreatedAt:     ep.CreationTimestamp.UTC().Format(time.RFC3339),
		Labels:        ep.Labels,
		Annotations:   ep.Annotations,
		ManagedFields: managedFields,
		Subsets:       subsets,
	}
}

func ListEndpoints(lister listerscorev1.EndpointsLister, namespace string) ([]dto.Endpoint, error) { //nolint:staticcheck
	var eps []*corev1.Endpoints //nolint:staticcheck
	var err error
	if namespace == "" {
		eps, err = lister.List(labels.Everything())
	} else {
		eps, err = lister.Endpoints(namespace).List(labels.Everything())
	}
	if err != nil {
		return nil, err
	}
	result := make([]dto.Endpoint, len(eps))
	for i, ep := range eps {
		result[i] = toEndpoint(ep)
	}
	return result, nil
}

func GetEndpointByName(lister listerscorev1.EndpointsLister, namespace, name string) (dto.Endpoint, error) { //nolint:staticcheck
	ep, err := lister.Endpoints(namespace).Get(name)
	if err != nil {
		return dto.Endpoint{}, err
	}
	return toEndpoint(ep), nil
}

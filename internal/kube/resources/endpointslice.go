package kubeResources

import (
	"time"

	"github.com/litelensapp/litelens/packages/core/dto"
	discoveryv1 "k8s.io/api/discovery/v1"
	"k8s.io/apimachinery/pkg/labels"
	discoveryv1listers "k8s.io/client-go/listers/discovery/v1"
	sigsyaml "sigs.k8s.io/yaml"
)

func toEndpointSlice(es *discoveryv1.EndpointSlice) dto.EndpointSlice {
	endpoints := make([]dto.EndpointSliceEndpoint, 0, len(es.Endpoints))
	for _, ep := range es.Endpoints {
		targetName := ""
		targetKind := ""
		if ep.TargetRef != nil {
			targetName = ep.TargetRef.Name
			targetKind = ep.TargetRef.Kind
		}
		endpoints = append(endpoints, dto.EndpointSliceEndpoint{
			Addresses:   ep.Addresses,
			Hostname:    derefString(ep.Hostname),
			NodeName:    derefString(ep.NodeName),
			Zone:        derefString(ep.Zone),
			TargetName:  targetName,
			TargetKind:  targetKind,
			Ready:       derefBool(ep.Conditions.Ready),
			Serving:     derefBool(ep.Conditions.Serving),
			Terminating: derefBool(ep.Conditions.Terminating),
		})
	}

	ports := make([]dto.EndpointSlicePort, 0, len(es.Ports))
	for _, p := range es.Ports {
		protocol := ""
		if p.Protocol != nil {
			protocol = string(*p.Protocol)
		}
		ports = append(ports, dto.EndpointSlicePort{
			Name:     derefString(p.Name),
			Port:     derefInt32(p.Port),
			Protocol: protocol,
		})
	}

	managedFields := make([]dto.ManagedField, 0, len(es.ManagedFields))
	for _, mf := range es.ManagedFields {
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

	lbl := es.Labels
	if lbl == nil {
		lbl = map[string]string{}
	}
	ann := es.Annotations
	if ann == nil {
		ann = map[string]string{}
	}

	controlledBy := ""
	serviceName := ""
	for _, ref := range es.OwnerReferences {
		if ref.Kind == "Service" {
			controlledBy = "Service " + ref.Name
			serviceName = ref.Name
			break
		}
	}

	return dto.EndpointSlice{
		Name:          es.Name,
		Namespace:     es.Namespace,
		AddressType:   string(es.AddressType),
		Ports:         ports,
		Endpoints:     endpoints,
		Age:           humanAge(es.CreationTimestamp.Time),
		CreatedAt:     es.CreationTimestamp.UTC().Format(time.RFC3339),
		Labels:        lbl,
		Annotations:   ann,
		ManagedFields: managedFields,
		ControlledBy:  controlledBy,
		ServiceName:   serviceName,
	}
}

func ListEndpointSlices(lister discoveryv1listers.EndpointSliceLister, namespaces []string) ([]dto.EndpointSlice, error) {
	ess, err := lister.List(labels.Everything())
	if err != nil {
		return nil, err
	}
	if len(namespaces) > 0 {
		nsSet := make(map[string]struct{}, len(namespaces))
		for _, ns := range namespaces {
			nsSet[ns] = struct{}{}
		}
		filtered := ess[:0:0]
		for _, es := range ess {
			if _, ok := nsSet[es.Namespace]; ok {
				filtered = append(filtered, es)
			}
		}
		ess = filtered
	}
	result := make([]dto.EndpointSlice, len(ess))
	for i, es := range ess {
		result[i] = toEndpointSlice(es)
	}
	return result, nil
}

func GetEndpointSliceByName(lister discoveryv1listers.EndpointSliceLister, namespace, name string) (dto.EndpointSlice, error) {
	es, err := lister.EndpointSlices(namespace).Get(name)
	if err != nil {
		return dto.EndpointSlice{}, err
	}
	return toEndpointSlice(es), nil
}

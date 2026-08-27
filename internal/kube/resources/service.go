package kubeResources

import (
	"fmt"
	"maps"
	"strings"
	"time"

	"github.com/litelensapp/litelens/packages/core/kube/dto"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/labels"
	listerscorev1 "k8s.io/client-go/listers/core/v1"
	sigsyaml "sigs.k8s.io/yaml"
)

func toService(svc *corev1.Service) dto.Service {
	externalIP := "-"
	if ingresses := svc.Status.LoadBalancer.Ingress; len(ingresses) > 0 {
		if ingresses[0].Hostname != "" {
			externalIP = ingresses[0].Hostname
		} else {
			externalIP = ingresses[0].IP
		}
	}

	portParts := make([]string, 0, len(svc.Spec.Ports))
	for _, p := range svc.Spec.Ports {
		targetStr := p.TargetPort.String()
		portStr := fmt.Sprintf("%d", p.Port)
		if targetStr != "" && targetStr != portStr {
			portParts = append(portParts, fmt.Sprintf("%d:%s/%s", p.Port, targetStr, p.Protocol))
		} else {
			portParts = append(portParts, fmt.Sprintf("%d/%s", p.Port, p.Protocol))
		}
	}

	selectorParts := make([]string, 0, len(svc.Spec.Selector))
	for k, v := range svc.Spec.Selector {
		selectorParts = append(selectorParts, fmt.Sprintf("%s=%s", k, v))
	}
	selector := strings.Join(selectorParts, ",")
	if selector == "" {
		selector = "-"
	}

	status := "Active"
	if svc.DeletionTimestamp != nil {
		status = "Terminating"
	}

	labels := make(map[string]string, len(svc.Labels))
	maps.Copy(labels, svc.Labels)
	annotations := make(map[string]string, len(svc.Annotations))
	maps.Copy(annotations, svc.Annotations)

	managedFields := make([]dto.ManagedField, 0, len(svc.ManagedFields))
	for _, mf := range svc.ManagedFields {
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

	sessionAffinity := string(svc.Spec.SessionAffinity)
	if sessionAffinity == "" {
		sessionAffinity = "None"
	}

	internalTrafficPolicy := "Cluster"
	if svc.Spec.InternalTrafficPolicy != nil {
		internalTrafficPolicy = string(*svc.Spec.InternalTrafficPolicy)
	}

	clusterIPs := make([]string, len(svc.Spec.ClusterIPs))
	copy(clusterIPs, svc.Spec.ClusterIPs)

	ipFamilyPolicy := ""
	if svc.Spec.IPFamilyPolicy != nil {
		ipFamilyPolicy = string(*svc.Spec.IPFamilyPolicy)
	}

	ipFamilies := make([]string, len(svc.Spec.IPFamilies))
	for i, f := range svc.Spec.IPFamilies {
		ipFamilies[i] = string(f)
	}

	servicePorts := make([]dto.ServicePort, len(svc.Spec.Ports))
	for i, p := range svc.Spec.Ports {
		servicePorts[i] = dto.ServicePort{
			Name:       p.Name,
			Port:       p.Port,
			TargetPort: p.TargetPort.String(),
			Protocol:   string(p.Protocol),
			NodePort:   p.NodePort,
		}
	}

	return dto.Service{
		Name:       svc.Name,
		Namespace:  svc.Namespace,
		Type:       string(svc.Spec.Type),
		ClusterIP:  svc.Spec.ClusterIP,
		Ports:      strings.Join(portParts, ", "),
		ExternalIP: externalIP,
		Selector:   selector,
		Age:        humanAge(svc.CreationTimestamp.Time),
		Status:     status,

		CreatedAt:             svc.CreationTimestamp.UTC().Format(time.RFC3339),
		Labels:                labels,
		Annotations:           annotations,
		ManagedFields:         managedFields,
		SessionAffinity:       sessionAffinity,
		InternalTrafficPolicy: internalTrafficPolicy,
		ClusterIPs:            clusterIPs,
		IPFamilyPolicy:        ipFamilyPolicy,
		IPFamilies:            ipFamilies,
		ServicePorts:          servicePorts,
	}
}

func GetServiceByName(lister listerscorev1.ServiceLister, namespace, name string) (dto.Service, error) {
	svc, err := lister.Services(namespace).Get(name)
	if err != nil {
		return dto.Service{}, err
	}
	return toService(svc), nil
}

func ListServices(lister listerscorev1.ServiceLister, namespaces []string) ([]dto.Service, error) {
	svcs, err := lister.List(labels.Everything())
	if err != nil {
		return nil, err
	}
	if len(namespaces) > 0 {
		nsSet := make(map[string]struct{}, len(namespaces))
		for _, ns := range namespaces {
			nsSet[ns] = struct{}{}
		}
		filtered := svcs[:0:0]
		for _, svc := range svcs {
			if _, ok := nsSet[svc.Namespace]; ok {
				filtered = append(filtered, svc)
			}
		}
		svcs = filtered
	}
	result := make([]dto.Service, len(svcs))
	for i, s := range svcs {
		result[i] = toService(s)
	}
	return result, nil
}

package kubeResources

import (
	"fmt"
	"maps"
	"time"

	"strings"

	"github.com/gknguyen/litelens/internal/dto"

	networkingv1 "k8s.io/api/networking/v1"
	"k8s.io/apimachinery/pkg/labels"
	listersnetworkingv1 "k8s.io/client-go/listers/networking/v1"
)

func toIngress(ing *networkingv1.Ingress) dto.Ingress {
	// Load balancers from status
	var lbs []string
	for _, lb := range ing.Status.LoadBalancer.Ingress {
		if lb.IP != "" {
			lbs = append(lbs, lb.IP)
		} else if lb.Hostname != "" {
			lbs = append(lbs, lb.Hostname)
		}
	}
	loadBalancers := strings.Join(lbs, ", ")
	if loadBalancers == "" {
		loadBalancers = "-"
	}

	var rules []dto.IngressRule
	for _, rule := range ing.Spec.Rules {
		host := rule.Host
		if host == "" {
			host = "*"
		}
		var paths []dto.IngressPath
		if rule.HTTP != nil {
			for _, p := range rule.HTTP.Paths {
				backend := ""
				if p.Backend.Service != nil {
					svc := p.Backend.Service
					port := svc.Port.Name
					if port == "" {
						port = fmt.Sprintf("%d", svc.Port.Number)
					}
					backend = fmt.Sprintf("%s:%s", svc.Name, port)
				}
				paths = append(paths, dto.IngressPath{Path: p.Path, Backend: backend})
			}
		}
		rules = append(rules, dto.IngressRule{Host: host, Paths: paths})
	}

	return dto.Ingress{
		Name:          ing.Name,
		Namespace:     ing.Namespace,
		LoadBalancers: loadBalancers,
		Rules:         rules,
		Age:           humanAge(ing.CreationTimestamp.Time),
	}
}

func ListIngresses(lister listersnetworkingv1.IngressLister, namespace string) ([]dto.Ingress, error) {
	var ings []*networkingv1.Ingress
	var err error
	if namespace == "" {
		ings, err = lister.List(labels.Everything())
	} else {
		ings, err = lister.Ingresses(namespace).List(labels.Everything())
	}
	if err != nil {
		return nil, err
	}
	result := make([]dto.Ingress, len(ings))
	for i, ing := range ings {
		result[i] = toIngress(ing)
	}
	return result, nil
}

func toIngressDetail(ing *networkingv1.Ingress) dto.IngressDetail {
	// Load balancers
	var lbs []string
	for _, lb := range ing.Status.LoadBalancer.Ingress {
		if lb.IP != "" {
			lbs = append(lbs, lb.IP)
		} else if lb.Hostname != "" {
			lbs = append(lbs, lb.Hostname)
		}
	}
	loadBalancers := strings.Join(lbs, ", ")

	// Labels + Annotations (use maps.Copy pattern)
	lbls := map[string]string{}
	maps.Copy(lbls, ing.Labels)
	annots := map[string]string{}
	maps.Copy(annots, ing.Annotations)

	// Structured rules + collect unique ports
	var rules []dto.IngressRule
	portSet := map[string]struct{}{}
	for _, rule := range ing.Spec.Rules {
		host := rule.Host
		if host == "" {
			host = "*"
		}
		var paths []dto.IngressPath
		if rule.HTTP != nil {
			for _, p := range rule.HTTP.Paths {
				backend := ""
				if p.Backend.Service != nil {
					svc := p.Backend.Service
					port := ""
					if svc.Port.Name != "" {
						port = svc.Port.Name
					} else {
						port = fmt.Sprintf("%d", svc.Port.Number)
					}
					backend = fmt.Sprintf("%s:%s", svc.Name, port)
					portSet[port] = struct{}{}
				}
				paths = append(paths, dto.IngressPath{
					Path:    p.Path,
					Backend: backend,
				})
			}
		}
		rules = append(rules, dto.IngressRule{Host: host, Paths: paths})
	}

	// Collect ports
	var portList []string
	for p := range portSet {
		portList = append(portList, p)
	}
	ports := strings.Join(portList, ", ")

	return dto.IngressDetail{
		Name:          ing.Name,
		Namespace:     ing.Namespace,
		Age:           humanAge(ing.CreationTimestamp.Time),
		CreatedAt:     ing.CreationTimestamp.UTC().Format(time.RFC3339),
		Labels:        lbls,
		Annotations:   annots,
		LoadBalancers: loadBalancers,
		Ports:         ports,
		Rules:         rules,
	}
}

func GetIngressByName(lister listersnetworkingv1.IngressLister, namespace, name string) (dto.IngressDetail, error) {
	ing, err := lister.Ingresses(namespace).Get(name)
	if err != nil {
		return dto.IngressDetail{}, err
	}
	return toIngressDetail(ing), nil
}

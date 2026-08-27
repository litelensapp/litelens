package kubeResources

import (
	"time"

	"github.com/litelensapp/litelens/packages/core/kube/dto"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/labels"
	listerscorev1 "k8s.io/client-go/listers/core/v1"
)

func toServiceAccount(sa *corev1.ServiceAccount) dto.ServiceAccount {
	secrets := make([]string, len(sa.Secrets))
	for i, s := range sa.Secrets {
		secrets[i] = s.Name
	}
	return dto.ServiceAccount{
		Name:      sa.Name,
		Namespace: sa.Namespace,
		Age:       humanAge(sa.CreationTimestamp.Time),
		CreatedAt: sa.CreationTimestamp.UTC().Format(time.RFC3339),
		Secrets:   secrets,
	}
}

func GetServiceAccountByName(saLister listerscorev1.ServiceAccountLister, namespace, name string) (dto.ServiceAccount, error) {
	sa, err := saLister.ServiceAccounts(namespace).Get(name)
	if err != nil {
		return dto.ServiceAccount{}, err
	}
	return toServiceAccount(sa), nil
}

func ListServiceAccounts(saLister listerscorev1.ServiceAccountLister, namespaces []string) ([]dto.ServiceAccount, error) {
	sas, err := saLister.List(labels.Everything())
	if err != nil {
		return nil, err
	}
	if len(namespaces) > 0 {
		nsSet := make(map[string]struct{}, len(namespaces))
		for _, ns := range namespaces {
			nsSet[ns] = struct{}{}
		}
		filtered := sas[:0:0]
		for _, sa := range sas {
			if _, ok := nsSet[sa.Namespace]; ok {
				filtered = append(filtered, sa)
			}
		}
		sas = filtered
	}
	result := make([]dto.ServiceAccount, len(sas))
	for i, sa := range sas {
		result[i] = toServiceAccount(sa)
	}
	return result, nil
}

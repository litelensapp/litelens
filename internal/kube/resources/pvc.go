package kubeResources

import (
	"fmt"
	"maps"
	"sort"
	"strings"
	"time"

	"github.com/litelensapp/litelens/internal/dto"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/labels"
	listerscorev1 "k8s.io/client-go/listers/core/v1"
)

func toPersistentVolumeClaim(pvc *corev1.PersistentVolumeClaim, podNames []string) dto.PersistentVolumeClaim {
	storageClass := "-"
	if pvc.Spec.StorageClassName != nil && *pvc.Spec.StorageClassName != "" {
		storageClass = *pvc.Spec.StorageClassName
	}

	size := "-"
	if q, ok := pvc.Spec.Resources.Requests[corev1.ResourceStorage]; ok {
		size = q.String()
	}

	status := string(pvc.Status.Phase)
	if pvc.DeletionTimestamp != nil {
		status = "Terminating"
	}

	pods := strings.Join(podNames, ", ")
	if pods == "" {
		pods = "-"
	}

	return dto.PersistentVolumeClaim{
		Name:         pvc.Name,
		Namespace:    pvc.Namespace,
		StorageClass: storageClass,
		Size:         size,
		Pods:         pods,
		Age:          humanAge(pvc.CreationTimestamp.Time),
		Status:       status,
	}
}

func ListPersistentVolumeClaims(
	pvcLister listerscorev1.PersistentVolumeClaimLister,
	podLister listerscorev1.PodLister,
	namespace string,
) ([]dto.PersistentVolumeClaim, error) {
	var pvcs []*corev1.PersistentVolumeClaim
	var err error
	if namespace == "" {
		pvcs, err = pvcLister.List(labels.Everything())
	} else {
		pvcs, err = pvcLister.PersistentVolumeClaims(namespace).List(labels.Everything())
	}
	if err != nil {
		return nil, err
	}

	// Build a map: namespace/claimName → []podName for fast lookup.
	claimToPods := map[string][]string{}
	var pods []*corev1.Pod
	if namespace == "" {
		pods, err = podLister.List(labels.Everything())
	} else {
		pods, err = podLister.Pods(namespace).List(labels.Everything())
	}
	if err == nil {
		for _, pod := range pods {
			for _, vol := range pod.Spec.Volumes {
				if vol.PersistentVolumeClaim != nil {
					key := pod.Namespace + "/" + vol.PersistentVolumeClaim.ClaimName
					claimToPods[key] = append(claimToPods[key], pod.Name)
				}
			}
		}
	}
	for k := range claimToPods {
		sort.Strings(claimToPods[k])
	}

	result := make([]dto.PersistentVolumeClaim, len(pvcs))
	for i, pvc := range pvcs {
		key := pvc.Namespace + "/" + pvc.Name
		result[i] = toPersistentVolumeClaim(pvc, claimToPods[key])
	}
	return result, nil
}

func GetPersistentVolumeClaimByName(
	pvcLister listerscorev1.PersistentVolumeClaimLister,
	podLister listerscorev1.PodLister,
	namespace, name string,
) (*dto.PersistentVolumeClaimDetail, error) {
	obj, err := pvcLister.PersistentVolumeClaims(namespace).Get(name)
	if err != nil {
		return nil, err
	}

	storageClass := "-"
	if obj.Spec.StorageClassName != nil && *obj.Spec.StorageClassName != "" {
		storageClass = *obj.Spec.StorageClassName
	}

	size := "-"
	if q, ok := obj.Spec.Resources.Requests[corev1.ResourceStorage]; ok {
		size = q.String()
	}

	status := string(obj.Status.Phase)
	if obj.DeletionTimestamp != nil {
		status = "Terminating"
	}

	labelsCopy := make(map[string]string, len(obj.Labels))
	maps.Copy(labelsCopy, obj.Labels)
	annotationsCopy := make(map[string]string, len(obj.Annotations))
	maps.Copy(annotationsCopy, obj.Annotations)

	finalizers := make([]string, len(obj.Finalizers))
	copy(finalizers, obj.Finalizers)

	accessModes := make([]string, len(obj.Spec.AccessModes))
	for i, am := range obj.Spec.AccessModes {
		accessModes[i] = string(am)
	}

	matchLabels := map[string]string{}
	var matchExprs []string
	if obj.Spec.Selector != nil {
		maps.Copy(matchLabels, obj.Spec.Selector.MatchLabels)
		for _, expr := range obj.Spec.Selector.MatchExpressions {
			matchExprs = append(matchExprs, fmt.Sprintf("%s %s %v", expr.Key, expr.Operator, expr.Values))
		}
	}

	// find pods that mount this PVC
	var podNames []string
	pods, _ := podLister.Pods(namespace).List(labels.Everything())
	for _, pod := range pods {
		for _, vol := range pod.Spec.Volumes {
			if vol.PersistentVolumeClaim != nil && vol.PersistentVolumeClaim.ClaimName == name {
				podNames = append(podNames, pod.Name)
				break
			}
		}
	}
	sort.Strings(podNames)
	if podNames == nil {
		podNames = []string{}
	}

	return &dto.PersistentVolumeClaimDetail{
		Name:         obj.Name,
		Namespace:    obj.Namespace,
		StorageClass: storageClass,
		Size:         size,
		Pods:         podNames,
		Age:          humanAge(obj.CreationTimestamp.Time),
		CreatedAt:    obj.CreationTimestamp.Time.Format(time.RFC3339),
		Status:       status,
		Labels:       labelsCopy,
		Annotations:  annotationsCopy,
		Finalizers:   finalizers,
		AccessModes:  accessModes,
		MatchLabels:  matchLabels,
		MatchExprs:   matchExprs,
	}, nil
}

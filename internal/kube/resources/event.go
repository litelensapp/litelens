package kubeResources

import (
	"github.com/gknguyen/litelens/internal/dto"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/labels"
	listerscorev1 "k8s.io/client-go/listers/core/v1"
	sigsyaml "sigs.k8s.io/yaml"
)

func toEvent(e *corev1.Event) dto.Event {
	lastSeen := ""
	lastSeenAt := int64(0)
	if !e.LastTimestamp.IsZero() {
		lastSeen = humanAge(e.LastTimestamp.Time)
		lastSeenAt = e.LastTimestamp.Unix()
	}

	firstSeen := ""
	firstSeenAt := e.CreationTimestamp.Unix()
	if !e.FirstTimestamp.IsZero() {
		firstSeen = humanAge(e.FirstTimestamp.Time)
		firstSeenAt = e.FirstTimestamp.Unix()
	}

	managedFields := make([]dto.ManagedField, 0, len(e.ManagedFields))
	for _, mf := range e.ManagedFields {
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

	return dto.Event{
		Type:               e.Type,
		Message:            e.Message,
		Namespace:          e.Namespace,
		InvolvedObjectKind: e.InvolvedObject.Kind,
		InvolvedObjectName: e.InvolvedObject.Name,
		Source:             e.Source.Component,
		Count:              e.Count,
		Age:                humanAge(e.CreationTimestamp.Time),
		LastSeen:           lastSeen,
		CreatedAt:          e.CreationTimestamp.Unix(),

		Name:                    e.Name,
		Reason:                  e.Reason,
		FirstSeen:               firstSeen,
		FirstSeenAt:             firstSeenAt,
		LastSeenAt:              lastSeenAt,
		InvolvedObjectFieldPath: e.InvolvedObject.FieldPath,
		InvolvedObjectNamespace: e.InvolvedObject.Namespace,
		ManagedFields:           managedFields,
	}
}

func GetEventByName(lister listerscorev1.EventLister, namespace, name string) (dto.Event, error) {
	e, err := lister.Events(namespace).Get(name)
	if err != nil {
		return dto.Event{}, err
	}
	return toEvent(e), nil
}

func ListEvents(lister listerscorev1.EventLister, namespace string) ([]dto.Event, error) {
	var events []*corev1.Event
	var err error
	if namespace == "" {
		events, err = lister.List(labels.Everything())
	} else {
		events, err = lister.Events(namespace).List(labels.Everything())
	}
	if err != nil {
		return nil, err
	}
	result := make([]dto.Event, len(events))
	for i, e := range events {
		result[i] = toEvent(e)
	}
	return result, nil
}

func ListWarningEvents(lister listerscorev1.EventLister, namespace string) ([]dto.Event, error) {
	events, err := ListEvents(lister, namespace)
	if err != nil {
		return nil, err
	}
	result := make([]dto.Event, 0, len(events))
	for _, e := range events {
		if e.Type == "Warning" {
			result = append(result, e)
		}
	}
	return result, nil
}

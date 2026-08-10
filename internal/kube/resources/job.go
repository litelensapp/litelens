package kubeResources

import (
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/litelensapp/litelens/internal/dto"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/labels"
	listersbatchv1 "k8s.io/client-go/listers/batch/v1"
	sigsyaml "sigs.k8s.io/yaml"
)

func humanDuration(d time.Duration) string {
	switch {
	case d >= 24*time.Hour:
		return fmt.Sprintf("%dd", int(d.Hours()/24))
	case d >= time.Hour:
		return fmt.Sprintf("%dh%dm", int(d.Hours()), int(d.Minutes())%60)
	case d >= time.Minute:
		return fmt.Sprintf("%dm%ds", int(d.Minutes()), int(d.Seconds())%60)
	default:
		return fmt.Sprintf("%ds", int(d.Seconds()))
	}
}

func toJob(j *batchv1.Job) dto.Job {
	var desired int32 = 1
	if j.Spec.Completions != nil {
		desired = *j.Spec.Completions
	}

	var parallelism int32 = 1
	if j.Spec.Parallelism != nil {
		parallelism = *j.Spec.Parallelism
	}

	var conditions []string
	status := "Unknown"
	for _, c := range j.Status.Conditions {
		if c.Status == corev1.ConditionTrue {
			conditions = append(conditions, string(c.Type))
			if status == "Unknown" {
				status = string(c.Type)
			}
		}
	}

	duration := ""
	if j.Status.StartTime != nil {
		end := time.Now()
		if j.Status.CompletionTime != nil {
			end = j.Status.CompletionTime.Time
		}
		duration = humanDuration(end.Sub(j.Status.StartTime.Time))
	}

	return dto.Job{
		Name:        j.Name,
		Namespace:   j.Namespace,
		Completions: desired,
		Age:         humanAge(j.CreationTimestamp.Time),
		Conditions:  conditions,
		Resumed:     j.Spec.Suspend == nil || !*j.Spec.Suspend,
		Status:      status,
		Succeeded:   j.Status.Succeeded,
		Parallelism: parallelism,
		Duration:    duration,

		CreatedAt: j.CreationTimestamp.Format(time.RFC3339),
		Labels: func() map[string]string {
			if j.Labels == nil {
				return map[string]string{}
			}
			return j.Labels
		}(),
		Annotations: func() map[string]string {
			if j.Annotations == nil {
				return map[string]string{}
			}
			return j.Annotations
		}(),
		ManagedFields: func() []dto.ManagedField {
			out := make([]dto.ManagedField, 0, len(j.ManagedFields))
			for _, mf := range j.ManagedFields {
				fieldsYAML := ""
				if raw := mf.FieldsV1.GetRawBytes(); len(raw) > 0 {
					if yamlBytes, err := sigsyaml.JSONToYAML(raw); err == nil {
						fieldsYAML = string(yamlBytes)
					}
				}
				out = append(out, dto.ManagedField{
					Manager:    mf.Manager,
					Operation:  string(mf.Operation),
					FieldsYAML: fieldsYAML,
				})
			}
			return out
		}(),
		Selector: func() string {
			if j.Spec.Selector == nil {
				return ""
			}
			keys := make([]string, 0, len(j.Spec.Selector.MatchLabels))
			for k := range j.Spec.Selector.MatchLabels {
				keys = append(keys, k)
			}
			sort.Strings(keys)
			parts := make([]string, 0, len(keys))
			for _, k := range keys {
				parts = append(parts, k+"="+j.Spec.Selector.MatchLabels[k])
			}
			return strings.Join(parts, ", ")
		}(),
		CompletionMode: func() string {
			if j.Spec.CompletionMode == nil {
				return "NonIndexed"
			}
			return string(*j.Spec.CompletionMode)
		}(),
		StartTime: func() string {
			if j.Status.StartTime == nil {
				return ""
			}
			return j.Status.StartTime.Format(time.RFC3339)
		}(),
		StartTimeAge: func() string {
			if j.Status.StartTime == nil {
				return ""
			}
			return humanAge(j.Status.StartTime.Time)
		}(),
		CompletedAt: func() string {
			if j.Status.CompletionTime == nil {
				return ""
			}
			return j.Status.CompletionTime.Format(time.RFC3339)
		}(),
		CompletedAtAge: func() string {
			if j.Status.CompletionTime == nil {
				return ""
			}
			return humanAge(j.Status.CompletionTime.Time)
		}(),
		PodsStatuses: fmt.Sprintf("%d Active (%d Ready) / %d Succeeded / %d Failed",
			j.Status.Active, j.Status.Ready, j.Status.Succeeded, j.Status.Failed),
		PodStatus: func() string {
			var parts []string
			if j.Status.Active > 0 {
				parts = append(parts, fmt.Sprintf("Active: %d", j.Status.Active))
			}
			if j.Status.Succeeded > 0 {
				parts = append(parts, fmt.Sprintf("Succeeded: %d", j.Status.Succeeded))
			}
			if j.Status.Failed > 0 {
				parts = append(parts, fmt.Sprintf("Failed: %d", j.Status.Failed))
			}
			if len(parts) == 0 {
				return ""
			}
			return strings.Join(parts, ", ")
		}(),
	}
}

func GetJobByName(lister listersbatchv1.JobLister, namespace, name string) (dto.Job, error) {
	job, err := lister.Jobs(namespace).Get(name)
	if err != nil {
		return dto.Job{}, err
	}
	return toJob(job), nil
}

func ListJobs(lister listersbatchv1.JobLister, namespace string) ([]dto.Job, error) {
	var jobs []*batchv1.Job
	var err error
	if namespace == "" {
		jobs, err = lister.List(labels.Everything())
	} else {
		jobs, err = lister.Jobs(namespace).List(labels.Everything())
	}
	if err != nil {
		return nil, err
	}
	result := make([]dto.Job, len(jobs))
	for i, j := range jobs {
		result[i] = toJob(j)
	}
	return result, nil
}

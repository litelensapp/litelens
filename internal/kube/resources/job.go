package kubeResources

import (
	"fmt"
	"log"
	"sort"
	"strings"
	"time"

	"github.com/litelensapp/litelens/packages/core/kube/dto"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/labels"
	listersbatchv1 "k8s.io/client-go/listers/batch/v1"
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

func isJobStatusConditionType(t batchv1.JobConditionType) bool {
	switch t {
	case batchv1.JobComplete, batchv1.JobFailed, batchv1.JobSuspended:
		return true
	default:
		return false
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

	var conditions []dto.JobCondition
	status := "Unknown"
	for _, c := range j.Status.Conditions {
		lastProbeTime := ""
		if !c.LastProbeTime.IsZero() {
			lastProbeTime = c.LastProbeTime.Format(time.RFC3339)
		}
		lastTransitionTime := ""
		if !c.LastTransitionTime.IsZero() {
			lastTransitionTime = c.LastTransitionTime.Format(time.RFC3339)
		}
		conditions = append(conditions, dto.JobCondition{
			Type:               string(c.Type),
			Status:             string(c.Status),
			Message:            c.Message,
			Reason:             c.Reason,
			LastProbeTime:      lastProbeTime,
			LastTransitionTime: lastTransitionTime,
		})
		if c.Status == corev1.ConditionTrue && status == "Unknown" && isJobStatusConditionType(c.Type) {
			status = string(c.Type)
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
		ManagedFields: toManagedFields(j),
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

func ListJobs(lister listersbatchv1.JobLister, namespaces []string) ([]dto.Job, error) {
	var jobs []*batchv1.Job
	if len(namespaces) == 0 {
		all, err := lister.List(labels.Everything())
		if err != nil {
			return nil, err
		}
		jobs = all
	} else {
		for _, ns := range namespaces {
			nsJobs, err := lister.Jobs(ns).List(labels.Everything())
			if err != nil {
				// Tolerate per-namespace errors (e.g., RBAC 403) but log them so
				// genuine failures (API server errors, etc.) remain visible.
				log.Printf("kubeResources: ListJobs: namespace %q: %v", ns, err)
				continue
			}
			jobs = append(jobs, nsJobs...)
		}
	}
	result := make([]dto.Job, len(jobs))
	for i, j := range jobs {
		result[i] = toJob(j)
	}
	return result, nil
}

func SummarizeJobs(jobs []*batchv1.Job) dto.JobSummary {
	summary := dto.JobSummary{}
	for _, j := range jobs {
		succeeded := false
		failed := false
		for _, c := range j.Status.Conditions {
			if c.Type == batchv1.JobComplete {
				succeeded = true
			}
			if c.Type == batchv1.JobFailed {
				failed = true
			}
		}
		if succeeded {
			summary.Succeeded++
		} else if failed {
			summary.Failed++
		} else {
			summary.Pending++
		}
	}
	return summary
}

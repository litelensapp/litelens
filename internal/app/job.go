package app

import (
	"context"
	"fmt"
	"log"

	kubeResources "github.com/litelensapp/litelens/internal/kube/resources"
	"github.com/litelensapp/litelens/packages/core/kube/dto"
	"github.com/wailsapp/wails/v2/pkg/runtime"
	batchv1 "k8s.io/api/batch/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	sigsyaml "sigs.k8s.io/yaml"
)

func (a *App) GetJobByName(namespace, name string) (dto.Job, error) {
	h := a.activeFactory()
	if !waitForResourceSync(h, "jobs") {
		return dto.Job{}, nil
	}
	result, err := kubeResources.GetJobByName(h.Factory.Batch().V1().Jobs().Lister(), namespace, name)
	if err != nil {
		log.Printf("app: GetJobByName: %v", err)
		return dto.Job{}, nil
	}
	return result, nil
}

func (a *App) ListJobs() ([]dto.Job, error) {
	h, namespaces := a.activeFactoryAndNamespaces()
	if !waitForResourceSync(h, "jobs") {
		return []dto.Job{}, nil
	}
	result, err := kubeResources.ListJobs(h.Factory.Batch().V1().Jobs().Lister(), namespaces)
	if err != nil {
		log.Printf("app: ListJobs: %v", err)
		return []dto.Job{}, nil
	}
	return result, nil
}

func (a *App) GetJobsSummary() (dto.JobSummary, error) {
	h, namespaces := a.activeFactoryAndNamespaces()
	if !waitForResourceSync(h, "jobs") {
		return dto.JobSummary{}, nil
	}
	lister := h.Factory.Batch().V1().Jobs().Lister()
	var jobs []*batchv1.Job
	if len(namespaces) == 0 {
		all, err := lister.List(labels.Everything())
		if err != nil {
			log.Printf("app: GetJobsSummary: %v", err)
			return dto.JobSummary{}, nil
		}
		jobs = all
	} else {
		for _, ns := range namespaces {
			nsJobs, err := lister.Jobs(ns).List(labels.Everything())
			if err != nil {
				// Tolerate per-namespace errors (e.g., RBAC 403) but log them so
				// genuine failures (API server errors, etc.) remain visible.
				log.Printf("app: GetJobsSummary: namespace %q: %v", ns, err)
				continue
			}
			jobs = append(jobs, nsJobs...)
		}
	}
	return kubeResources.SummarizeJobs(jobs), nil
}

func (a *App) emitJobs() {
	h, namespaces := a.activeFactoryAndNamespaces()
	if !waitForResourceSync(h, "jobs") {
		return
	}
	lister := h.Factory.Batch().V1().Jobs().Lister()
	data, err := kubeResources.ListJobs(lister, namespaces)
	if err != nil {
		log.Printf("app: emitJobs: %v", err)
		return
	}
	runtime.EventsEmit(a.ctx, "jobs:update", data)
}

// DeleteJob deletes a Job from the specified namespace.
func (a *App) DeleteJob(namespace, name string) error {
	cs, err := a.activeClientset()
	if err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(context.Background(), apiMutationTimeout)
	defer cancel()
	err = cs.BatchV1().Jobs(namespace).Delete(ctx, name, metav1.DeleteOptions{})
	if err != nil && !errors.IsNotFound(err) {
		return fmt.Errorf("delete Job: %w", err)
	}

	// Emit update event after successful delete
	a.emitJobs()

	return nil
}

// DeleteJobs deletes multiple Jobs, handling best-effort deletion across namespaces.
func (a *App) DeleteJobs(items []dto.JobRef) error {
	cs, err := a.activeClientset()
	if err != nil {
		return err
	}

	err = deleteRefsBestEffort(items,
		func(r dto.JobRef) string { return r.Namespace },
		func(r dto.JobRef) string { return r.Name },
		"jobs",
		func(ctx context.Context, namespace, name string) error {
			return cs.BatchV1().Jobs(namespace).Delete(ctx, name, metav1.DeleteOptions{})
		},
	)

	a.emitJobs()

	return err
}

func (a *App) GetJobYAML(namespace, name string) (string, error) {
	cs, err := a.activeClientset()
	if err != nil {
		return "", err
	}

	ctx, cancel := context.WithTimeout(context.Background(), apiReadTimeout)
	defer cancel()
	job, err := cs.BatchV1().Jobs(namespace).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		return "", fmt.Errorf("get Job: %w", err)
	}

	yamlBytes, err := sigsyaml.Marshal(job)
	if err != nil {
		return "", fmt.Errorf("marshal Job to YAML: %w", err)
	}

	return string(yamlBytes), nil
}

func (a *App) UpdateJobYAML(namespace, yamlString string) error {
	cs, err := a.activeClientset()
	if err != nil {
		return err
	}

	var job batchv1.Job
	err = sigsyaml.Unmarshal([]byte(yamlString), &job)
	if err != nil {
		return fmt.Errorf("unmarshal YAML to Job: %w", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), apiMutationTimeout)
	defer cancel()
	_, err = cs.BatchV1().Jobs(namespace).Update(ctx, &job, metav1.UpdateOptions{})
	if err != nil {
		return fmt.Errorf("update Job: %w", err)
	}

	a.emitJobs()

	return nil
}

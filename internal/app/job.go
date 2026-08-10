package app

import (
	"context"
	"fmt"
	"log"
	"strings"

	"github.com/gknguyen/litelens/internal/dto"
	"github.com/gknguyen/litelens/internal/kube/resources"
	"github.com/wailsapp/wails/v2/pkg/runtime"
	batchv1 "k8s.io/api/batch/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	sigsyaml "sigs.k8s.io/yaml"
)

func (a *App) GetJobByName(namespace, name string) (dto.Job, error) {
	a.mu.RLock()
	h := a.factories[a.activeContext]
	a.mu.RUnlock()
	if h == nil {
		return dto.Job{}, nil
	}
	if h.IsForbidden("jobs") {
		return dto.Job{}, nil
	}
	<-h.GetSyncedChan("jobs")
	if h.IsForbidden("jobs") {
		return dto.Job{}, nil
	}
	result, err := kubeResources.GetJobByName(h.Factory.Batch().V1().Jobs().Lister(), namespace, name)
	if err != nil {
		log.Printf("app: GetJobByName: %v", err)
		return dto.Job{}, nil
	}
	return result, nil
}

func (a *App) ListJobs(namespace string) ([]dto.Job, error) {
	a.mu.RLock()
	h := a.factories[a.activeContext]
	a.mu.RUnlock()
	if h == nil {
		return []dto.Job{}, nil
	}
	if h.IsForbidden("jobs") {
		return []dto.Job{}, nil
	}
	<-h.GetSyncedChan("jobs")
	if h.IsForbidden("jobs") {
		return nil, nil
	}
	result, err := kubeResources.ListJobs(h.Factory.Batch().V1().Jobs().Lister(), namespace)
	if err != nil {
		log.Printf("app: ListJobs: %v", err)
		return []dto.Job{}, nil
	}
	return result, nil
}

func (a *App) emitJobs(namespace string) {
	a.mu.RLock()
	h := a.factories[a.activeContext]
	a.mu.RUnlock()
	if h == nil {
		return
	}
	if h.IsForbidden("jobs") {
		return
	}
	<-h.GetSyncedChan("jobs")
	if h.IsForbidden("jobs") {
		return
	}
	lister := h.Factory.Batch().V1().Jobs().Lister()
	allData, err := kubeResources.ListJobs(lister, "")
	if err != nil {
		log.Printf("app: emitJobs: %v", err)
		return
	}
	runtime.EventsEmit(a.ctx, "jobs:update", allData)
	if namespace != "" {
		nsData, err := kubeResources.ListJobs(lister, namespace)
		if err != nil {
			log.Printf("app: emitJobs ns=%s: %v", namespace, err)
			return
		}
		runtime.EventsEmit(a.ctx, "jobs:"+namespace+":update", nsData)
	}
}

// DeleteJob deletes a Job from the specified namespace.
func (a *App) DeleteJob(namespace, name string) error {
	a.mu.RLock()
	cs := a.clients[a.activeContext]
	a.mu.RUnlock()
	if cs == nil {
		return fmt.Errorf("not connected")
	}

	ctx, cancel := context.WithTimeout(context.Background(), apiMutationTimeout)
	defer cancel()
	err := cs.BatchV1().Jobs(namespace).Delete(ctx, name, metav1.DeleteOptions{})
	if err != nil && !errors.IsNotFound(err) {
		return fmt.Errorf("delete Job: %w", err)
	}

	// Emit update event after successful delete
	a.emitJobs(namespace)

	return nil
}

// DeleteJobs deletes multiple Jobs, handling best-effort deletion across namespaces.
func (a *App) DeleteJobs(items []dto.JobRef) error {
	a.mu.RLock()
	cs := a.clients[a.activeContext]
	a.mu.RUnlock()
	if cs == nil {
		return fmt.Errorf("not connected")
	}

	var msgs []string
	namespaces := make(map[string]bool)

	for _, ref := range items {
		ctx, cancel := context.WithTimeout(context.Background(), apiMutationTimeout)
		err := cs.BatchV1().Jobs(ref.Namespace).Delete(ctx, ref.Name, metav1.DeleteOptions{})
		cancel()
		if err != nil && !errors.IsNotFound(err) {
			msgs = append(msgs, fmt.Sprintf("%s/%s: %v", ref.Namespace, ref.Name, err))
		}
		namespaces[ref.Namespace] = true
	}

	// Emit updates for each unique namespace touched
	for ns := range namespaces {
		a.emitJobs(ns)
	}

	if len(msgs) > 0 {
		return fmt.Errorf("failed to delete %d of %d jobs: %s", len(msgs), len(items), strings.Join(msgs, "; "))
	}
	return nil
}

func (a *App) GetJobYAML(namespace, name string) (string, error) {
	a.mu.RLock()
	cs := a.clients[a.activeContext]
	a.mu.RUnlock()
	if cs == nil {
		return "", fmt.Errorf("not connected")
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
	a.mu.RLock()
	cs := a.clients[a.activeContext]
	a.mu.RUnlock()
	if cs == nil {
		return fmt.Errorf("not connected")
	}

	var job batchv1.Job
	err := sigsyaml.Unmarshal([]byte(yamlString), &job)
	if err != nil {
		return fmt.Errorf("unmarshal YAML to Job: %w", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), apiMutationTimeout)
	defer cancel()
	_, err = cs.BatchV1().Jobs(namespace).Update(ctx, &job, metav1.UpdateOptions{})
	if err != nil {
		return fmt.Errorf("update Job: %w", err)
	}

	a.emitJobs(namespace)

	return nil
}

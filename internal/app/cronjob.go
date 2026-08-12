package app

import (
	"context"
	"fmt"
	"log"
	"strings"

	"github.com/litelensapp/litelens/internal/dto"
	"github.com/litelensapp/litelens/internal/kube/resources"
	"github.com/wailsapp/wails/v2/pkg/runtime"
	batchv1 "k8s.io/api/batch/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	sigsyaml "sigs.k8s.io/yaml"
)

func (a *App) GetCronJobByName(namespace, name string) (dto.CronJob, error) {
	a.mu.RLock()
	h := a.factories[a.activeContext]
	a.mu.RUnlock()
	if h == nil {
		return dto.CronJob{}, nil
	}
	if h.IsForbidden("cronjobs") {
		return dto.CronJob{}, nil
	}
	<-h.GetSyncedChan("cronjobs")
	if h.IsForbidden("cronjobs") {
		return dto.CronJob{}, nil
	}
	result, err := kubeResources.GetCronJobByName(h.Factory.Batch().V1().CronJobs().Lister(), namespace, name)
	if err != nil {
		log.Printf("app: GetCronJobByName: %v", err)
		return dto.CronJob{}, nil
	}
	return result, nil
}

func (a *App) ListCronJobs(namespace string) ([]dto.CronJob, error) {
	a.mu.RLock()
	h := a.factories[a.activeContext]
	a.mu.RUnlock()
	if h == nil {
		return []dto.CronJob{}, nil
	}
	if h.IsForbidden("cronjobs") {
		return []dto.CronJob{}, nil
	}
	<-h.GetSyncedChan("cronjobs")
	if h.IsForbidden("cronjobs") {
		return nil, nil
	}
	result, err := kubeResources.ListCronJobs(h.Factory.Batch().V1().CronJobs().Lister(), namespace)
	if err != nil {
		log.Printf("app: ListCronJobs: %v", err)
		return []dto.CronJob{}, nil
	}
	return result, nil
}

func (a *App) emitCronJobs(namespace string) {
	a.mu.RLock()
	h := a.factories[a.activeContext]
	a.mu.RUnlock()
	if h == nil {
		return
	}
	if h.IsForbidden("cronjobs") {
		return
	}
	<-h.GetSyncedChan("cronjobs")
	if h.IsForbidden("cronjobs") {
		return
	}
	lister := h.Factory.Batch().V1().CronJobs().Lister()
	allData, err := kubeResources.ListCronJobs(lister, "")
	if err != nil {
		log.Printf("app: emitCronJobs: %v", err)
		return
	}
	runtime.EventsEmit(a.ctx, "cronjobs:update", allData)
	if namespace != "" {
		// Filter already-fetched cluster-wide data instead of re-listing
		nsData := make([]dto.CronJob, 0)
		for _, item := range allData {
			if item.Namespace == namespace {
				nsData = append(nsData, item)
			}
		}
		runtime.EventsEmit(a.ctx, "cronjobs:"+namespace+":update", nsData)
	}
}

// DeleteCronJob deletes a CronJob from the specified namespace.
func (a *App) DeleteCronJob(namespace, name string) error {
	a.mu.RLock()
	cs := a.clients[a.activeContext]
	a.mu.RUnlock()
	if cs == nil {
		return fmt.Errorf("not connected")
	}

	ctx, cancel := context.WithTimeout(context.Background(), apiMutationTimeout)
	defer cancel()
	err := cs.BatchV1().CronJobs(namespace).Delete(ctx, name, metav1.DeleteOptions{})
	if err != nil && !errors.IsNotFound(err) {
		return fmt.Errorf("delete CronJob: %w", err)
	}

	// Emit update event after successful delete
	a.emitCronJobs(namespace)

	return nil
}

// DeleteCronJobs deletes multiple CronJobs, handling best-effort deletion across namespaces.
func (a *App) DeleteCronJobs(items []dto.CronJobRef) error {
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
		err := cs.BatchV1().CronJobs(ref.Namespace).Delete(ctx, ref.Name, metav1.DeleteOptions{})
		cancel()
		if err != nil && !errors.IsNotFound(err) {
			msgs = append(msgs, fmt.Sprintf("%s/%s: %v", ref.Namespace, ref.Name, err))
		}
		namespaces[ref.Namespace] = true
	}

	// Emit updates for each unique namespace touched
	for ns := range namespaces {
		a.emitCronJobs(ns)
	}

	if len(msgs) > 0 {
		return fmt.Errorf("failed to delete %d of %d cronjobs: %s", len(msgs), len(items), strings.Join(msgs, "; "))
	}
	return nil
}

func (a *App) GetCronJobYAML(namespace, name string) (string, error) {
	a.mu.RLock()
	cs := a.clients[a.activeContext]
	a.mu.RUnlock()
	if cs == nil {
		return "", fmt.Errorf("not connected")
	}

	ctx, cancel := context.WithTimeout(context.Background(), apiReadTimeout)
	defer cancel()
	cj, err := cs.BatchV1().CronJobs(namespace).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		return "", fmt.Errorf("get CronJob: %w", err)
	}

	yamlBytes, err := sigsyaml.Marshal(cj)
	if err != nil {
		return "", fmt.Errorf("marshal CronJob to YAML: %w", err)
	}

	return string(yamlBytes), nil
}

func (a *App) UpdateCronJobYAML(namespace, yamlString string) error {
	a.mu.RLock()
	cs := a.clients[a.activeContext]
	a.mu.RUnlock()
	if cs == nil {
		return fmt.Errorf("not connected")
	}

	var cj batchv1.CronJob
	err := sigsyaml.Unmarshal([]byte(yamlString), &cj)
	if err != nil {
		return fmt.Errorf("unmarshal YAML to CronJob: %w", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), apiMutationTimeout)
	defer cancel()
	_, err = cs.BatchV1().CronJobs(namespace).Update(ctx, &cj, metav1.UpdateOptions{})
	if err != nil {
		return fmt.Errorf("update CronJob: %w", err)
	}

	a.emitCronJobs(namespace)

	return nil
}

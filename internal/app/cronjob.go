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

func (a *App) GetCronJobByName(namespace, name string) (dto.CronJob, error) {
	h := a.activeFactory()
	if !waitForResourceSync(h, "cronjobs") {
		return dto.CronJob{}, nil
	}
	result, err := kubeResources.GetCronJobByName(h.Factory.Batch().V1().CronJobs().Lister(), namespace, name)
	if err != nil {
		log.Printf("app: GetCronJobByName: %v", err)
		return dto.CronJob{}, nil
	}
	return result, nil
}

func (a *App) ListCronJobs() ([]dto.CronJob, error) {
	h, namespaces := a.activeFactoryAndNamespaces()
	if !waitForResourceSync(h, "cronjobs") {
		return []dto.CronJob{}, nil
	}
	result, err := kubeResources.ListCronJobs(h.Factory.Batch().V1().CronJobs().Lister(), namespaces)
	if err != nil {
		log.Printf("app: ListCronJobs: %v", err)
		return []dto.CronJob{}, nil
	}
	return result, nil
}

func (a *App) GetCronJobsSummary() (dto.CronJobSummary, error) {
	h, namespaces := a.activeFactoryAndNamespaces()
	if !waitForResourceSync(h, "cronjobs") {
		return dto.CronJobSummary{}, nil
	}
	lister := h.Factory.Batch().V1().CronJobs().Lister()
	cjs, err := lister.List(labels.Everything())
	if err != nil {
		log.Printf("app: GetCronJobsSummary: %v", err)
		return dto.CronJobSummary{}, nil
	}
	if len(namespaces) > 0 {
		nsSet := make(map[string]struct{}, len(namespaces))
		for _, ns := range namespaces {
			nsSet[ns] = struct{}{}
		}
		filtered := cjs[:0:0]
		for _, cj := range cjs {
			if _, ok := nsSet[cj.Namespace]; ok {
				filtered = append(filtered, cj)
			}
		}
		cjs = filtered
	}
	return kubeResources.SummarizeCronJobs(cjs), nil
}

func (a *App) emitCronJobs() {
	h, namespaces := a.activeFactoryAndNamespaces()
	if !waitForResourceSync(h, "cronjobs") {
		return
	}
	lister := h.Factory.Batch().V1().CronJobs().Lister()
	data, err := kubeResources.ListCronJobs(lister, namespaces)
	if err != nil {
		log.Printf("app: emitCronJobs: %v", err)
		return
	}
	runtime.EventsEmit(a.ctx, "cronjobs:update", data)
}

// DeleteCronJob deletes a CronJob from the specified namespace.
func (a *App) DeleteCronJob(namespace, name string) error {
	cs, err := a.activeClientset()
	if err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(context.Background(), apiMutationTimeout)
	defer cancel()
	err = cs.BatchV1().CronJobs(namespace).Delete(ctx, name, metav1.DeleteOptions{})
	if err != nil && !errors.IsNotFound(err) {
		return fmt.Errorf("delete CronJob: %w", err)
	}

	// Emit update event after successful delete
	a.emitCronJobs()

	return nil
}

// DeleteCronJobs deletes multiple CronJobs, handling best-effort deletion across namespaces.
func (a *App) DeleteCronJobs(items []dto.CronJobRef) error {
	cs, err := a.activeClientset()
	if err != nil {
		return err
	}

	err = deleteRefsBestEffort(items,
		func(r dto.CronJobRef) string { return r.Namespace },
		func(r dto.CronJobRef) string { return r.Name },
		"cronjobs",
		func(ctx context.Context, namespace, name string) error {
			return cs.BatchV1().CronJobs(namespace).Delete(ctx, name, metav1.DeleteOptions{})
		},
	)

	a.emitCronJobs()

	return err
}

func (a *App) GetCronJobYAML(namespace, name string) (string, error) {
	cs, err := a.activeClientset()
	if err != nil {
		return "", err
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
	cs, err := a.activeClientset()
	if err != nil {
		return err
	}

	var cj batchv1.CronJob
	err = sigsyaml.Unmarshal([]byte(yamlString), &cj)
	if err != nil {
		return fmt.Errorf("unmarshal YAML to CronJob: %w", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), apiMutationTimeout)
	defer cancel()
	_, err = cs.BatchV1().CronJobs(namespace).Update(ctx, &cj, metav1.UpdateOptions{})
	if err != nil {
		return fmt.Errorf("update CronJob: %w", err)
	}

	a.emitCronJobs()

	return nil
}

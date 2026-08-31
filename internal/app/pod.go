package app

import (
	"context"
	"fmt"
	"log"

	"github.com/litelensapp/litelens/internal/kube"
	kubeResources "github.com/litelensapp/litelens/internal/kube/resources"
	"github.com/litelensapp/litelens/packages/core/kube/dto"
	"github.com/wailsapp/wails/v2/pkg/runtime"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	sigsyaml "sigs.k8s.io/yaml"
)

func (a *App) ListPods() ([]dto.Pod, error) {
	h, namespaces, mc := a.activeFactoryNamespacesAndMetrics()
	if !waitForResourceSync(h, "pods") {
		return []dto.Pod{}, nil
	}
	pods, err := kubeResources.ListPods(h.Factory.Core().V1().Pods().Lister(), namespaces)
	if err != nil {
		log.Printf("app: ListPods: %v", err)
		return []dto.Pod{}, nil
	}
	if mc != nil {
		ctx, cancel := context.WithTimeout(context.Background(), metricsFetchTimeout)
		defer cancel()
		metricsNamespace := ""
		if len(namespaces) == 1 {
			metricsNamespace = namespaces[0]
		}
		usage := kube.FetchPodMetrics(ctx, mc, metricsNamespace)
		pods = kubeResources.ApplyPodMetrics(pods, usage)
	}
	return pods, nil
}

func (a *App) GetPodByName(namespace, name string) (dto.Pod, error) {
	h := a.activeFactory()
	if !waitForResourceSync(h, "pods") {
		return dto.Pod{}, nil
	}
	result, err := kubeResources.GetPodByName(h.Factory.Core().V1().Pods().Lister(), namespace, name)
	if err != nil {
		log.Printf("app: GetPodByName: %v", err)
		return dto.Pod{}, nil
	}
	return result, nil
}

func (a *App) GetPodsSummary() (dto.PodSummary, error) {
	h, namespaces := a.activeFactoryAndNamespaces()
	if !waitForResourceSync(h, "pods") {
		return dto.PodSummary{}, nil
	}
	lister := h.Factory.Core().V1().Pods().Lister()
	var pods []*corev1.Pod
	if len(namespaces) == 0 {
		all, err := lister.List(labels.Everything())
		if err != nil {
			log.Printf("app: GetPodsSummary: %v", err)
			return dto.PodSummary{}, nil
		}
		pods = all
	} else {
		for _, ns := range namespaces {
			nsPods, err := lister.Pods(ns).List(labels.Everything())
			if err != nil {
				// Tolerate per-namespace errors (e.g., RBAC 403) but log them so
				// genuine failures (API server errors, etc.) remain visible.
				log.Printf("app: GetPodsSummary: namespace %q: %v", ns, err)
				continue
			}
			pods = append(pods, nsPods...)
		}
	}
	return kubeResources.SummarizePods(pods), nil
}

func (a *App) emitPods() {
	a.emitPodsWithMetrics(nil)
}

// emitPodsWithMetrics emits a pod update filtered by the currently active namespace
// selection (a.activeNamespaces). If allMetrics is nil, metrics are fetched
// asynchronously to avoid blocking the initial emit.
func (a *App) emitPodsWithMetrics(allMetrics map[string]dto.PodUsage) {
	h, namespaces, mc := a.activeFactoryNamespacesAndMetrics()
	if h == nil || h.IsForbidden("pods") {
		return
	}
	lister := h.Factory.Core().V1().Pods().Lister()

	pods, err := kubeResources.ListPods(lister, namespaces)
	if err != nil {
		log.Printf("app: emitPods: %v", err)
		return
	}

	if allMetrics != nil {
		pods = kubeResources.ApplyPodMetrics(pods, allMetrics)
	}
	runtime.EventsEmit(a.ctx, "pods:update", pods)

	if allMetrics == nil && mc != nil {
		go func() {
			ctx, cancel := context.WithTimeout(context.Background(), metricsFetchTimeout)
			defer cancel()
			metricsNamespace := ""
			if len(namespaces) == 1 {
				metricsNamespace = namespaces[0]
			}
			fetchedMetrics := kube.FetchPodMetrics(ctx, mc, metricsNamespace)
			if fetchedMetrics != nil {
				podsWithMetrics := kubeResources.ApplyPodMetrics(pods, fetchedMetrics)
				runtime.EventsEmit(a.ctx, "pods:update", podsWithMetrics)
			}
		}()
	}
}

// DeletePod deletes a Pod from the specified namespace.
func (a *App) DeletePod(namespace, name string) error {
	cs, err := a.activeClientset()
	if err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(context.Background(), apiMutationTimeout)
	defer cancel()
	err = cs.CoreV1().Pods(namespace).Delete(ctx, name, metav1.DeleteOptions{})
	if err != nil && !errors.IsNotFound(err) {
		return fmt.Errorf("delete Pod: %w", err)
	}

	a.emitPods()
	return nil
}

// DeletePods deletes multiple Pods, handling best-effort deletion across namespaces.
func (a *App) DeletePods(items []dto.PodRef) error {
	cs, err := a.activeClientset()
	if err != nil {
		return err
	}

	err = deleteRefsBestEffort(items,
		func(r dto.PodRef) string { return r.Namespace },
		func(r dto.PodRef) string { return r.Name },
		"pods",
		func(ctx context.Context, namespace, name string) error {
			return cs.CoreV1().Pods(namespace).Delete(ctx, name, metav1.DeleteOptions{})
		},
	)

	a.emitPods()

	return err
}

func (a *App) GetPodYAML(namespace, name string) (string, error) {
	cs, err := a.activeClientset()
	if err != nil {
		return "", err
	}

	ctx, cancel := context.WithTimeout(context.Background(), apiReadTimeout)
	defer cancel()
	pod, err := cs.CoreV1().Pods(namespace).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		return "", fmt.Errorf("get Pod: %w", err)
	}

	yamlBytes, err := sigsyaml.Marshal(pod)
	if err != nil {
		return "", fmt.Errorf("marshal Pod to YAML: %w", err)
	}

	return string(yamlBytes), nil
}

func (a *App) UpdatePodYAML(namespace, yamlString string) error {
	cs, err := a.activeClientset()
	if err != nil {
		return err
	}

	var pod corev1.Pod
	err = sigsyaml.Unmarshal([]byte(yamlString), &pod)
	if err != nil {
		return fmt.Errorf("unmarshal YAML to Pod: %w", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), apiMutationTimeout)
	defer cancel()
	_, err = cs.CoreV1().Pods(namespace).Update(ctx, &pod, metav1.UpdateOptions{})
	if err != nil {
		return fmt.Errorf("update Pod: %w", err)
	}

	a.emitPods()

	return nil
}

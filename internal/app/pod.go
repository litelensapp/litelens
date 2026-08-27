package app

import (
	"context"
	"fmt"
	"log"
	"strings"

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
	a.mu.RLock()
	h := a.factories[a.activeContext]
	mc := a.metricsClients[a.activeContext]
	namespaces := a.activeNamespaces
	a.mu.RUnlock()
	if h == nil {
		return []dto.Pod{}, nil
	}
	if h.IsForbidden("pods") {
		return []dto.Pod{}, nil
	}
	<-h.GetSyncedChan("pods")
	if h.IsForbidden("pods") {
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
	a.mu.RLock()
	h := a.factories[a.activeContext]
	a.mu.RUnlock()
	if h == nil {
		return dto.Pod{}, nil
	}
	if h.IsForbidden("pods") {
		return dto.Pod{}, nil
	}
	<-h.GetSyncedChan("pods")
	if h.IsForbidden("pods") {
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
	a.mu.RLock()
	h := a.factories[a.activeContext]
	namespaces := a.activeNamespaces
	a.mu.RUnlock()
	if h == nil {
		return dto.PodSummary{}, nil
	}
	if h.IsForbidden("pods") {
		return dto.PodSummary{}, nil
	}
	<-h.GetSyncedChan("pods")
	if h.IsForbidden("pods") {
		return dto.PodSummary{}, nil
	}
	lister := h.Factory.Core().V1().Pods().Lister()
	pods, err := lister.List(labels.Everything())
	if err != nil {
		log.Printf("app: GetPodsSummary: %v", err)
		return dto.PodSummary{}, nil
	}
	if len(namespaces) > 0 {
		nsSet := make(map[string]struct{}, len(namespaces))
		for _, ns := range namespaces {
			nsSet[ns] = struct{}{}
		}
		filtered := pods[:0:0]
		for _, p := range pods {
			if _, ok := nsSet[p.Namespace]; ok {
				filtered = append(filtered, p)
			}
		}
		pods = filtered
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
	a.mu.RLock()
	h := a.factories[a.activeContext]
	mc := a.metricsClients[a.activeContext]
	namespaces := a.activeNamespaces
	a.mu.RUnlock()
	if h == nil {
		return
	}
	if h.IsForbidden("pods") {
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
	a.mu.RLock()
	cs := a.clients[a.activeContext]
	a.mu.RUnlock()
	if cs == nil {
		return fmt.Errorf("not connected")
	}

	ctx, cancel := context.WithTimeout(context.Background(), apiMutationTimeout)
	defer cancel()
	err := cs.CoreV1().Pods(namespace).Delete(ctx, name, metav1.DeleteOptions{})
	if err != nil && !errors.IsNotFound(err) {
		return fmt.Errorf("delete Pod: %w", err)
	}

	a.emitPods()
	return nil
}

// DeletePods deletes multiple Pods, handling best-effort deletion across namespaces.
func (a *App) DeletePods(items []dto.PodRef) error {
	a.mu.RLock()
	cs := a.clients[a.activeContext]
	a.mu.RUnlock()
	if cs == nil {
		return fmt.Errorf("not connected")
	}

	var msgs []string

	for _, ref := range items {
		ctx, cancel := context.WithTimeout(context.Background(), apiMutationTimeout)
		err := cs.CoreV1().Pods(ref.Namespace).Delete(ctx, ref.Name, metav1.DeleteOptions{})
		cancel()
		if err != nil && !errors.IsNotFound(err) {
			msgs = append(msgs, fmt.Sprintf("%s/%s: %v", ref.Namespace, ref.Name, err))
		}
	}

	a.emitPods()

	if len(msgs) > 0 {
		return fmt.Errorf("failed to delete %d of %d pods: %s", len(msgs), len(items), strings.Join(msgs, "; "))
	}
	return nil
}

func (a *App) GetPodYAML(namespace, name string) (string, error) {
	a.mu.RLock()
	cs := a.clients[a.activeContext]
	a.mu.RUnlock()
	if cs == nil {
		return "", fmt.Errorf("not connected")
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
	a.mu.RLock()
	cs := a.clients[a.activeContext]
	a.mu.RUnlock()
	if cs == nil {
		return fmt.Errorf("not connected")
	}

	var pod corev1.Pod
	err := sigsyaml.Unmarshal([]byte(yamlString), &pod)
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

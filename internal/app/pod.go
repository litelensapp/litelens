package app

import (
	"context"
	"fmt"
	"log"
	"strings"

	"github.com/litelensapp/litelens/packages/core/dto"
	"github.com/litelensapp/litelens/internal/kube"
	kubeResources "github.com/litelensapp/litelens/internal/kube/resources"
	"github.com/wailsapp/wails/v2/pkg/runtime"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	sigsyaml "sigs.k8s.io/yaml"
)

func (a *App) ListPods(namespaces []string) ([]dto.Pod, error) {
	a.mu.RLock()
	h := a.factories[a.activeContext]
	mc := a.metricsClients[a.activeContext]
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

func (a *App) GetPodsSummary(namespace string) (dto.PodSummary, error) {
	a.mu.RLock()
	h := a.factories[a.activeContext]
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
	var pods []*corev1.Pod
	var err error
	lister := h.Factory.Core().V1().Pods().Lister()
	if namespace == "" {
		pods, err = lister.List(labels.Everything())
	} else {
		pods, err = lister.Pods(namespace).List(labels.Everything())
	}
	if err != nil {
		log.Printf("app: GetPodsSummary: %v", err)
		return dto.PodSummary{}, nil
	}
	return kubeResources.SummarizePods(pods), nil
}

func (a *App) emitPods(namespaces []string) {
	a.emitPodsWithMetrics(namespaces, nil)
}

// emitPodsWithMetrics emits pod updates with optional pre-fetched cluster-wide metrics.
// If allMetrics is nil, metrics are fetched asynchronously to avoid blocking the initial emit.
// This variant avoids redundant metric fetches when emitting updates for multiple namespaces.
func (a *App) emitPodsWithMetrics(namespaces []string, allMetrics map[string]dto.PodUsage) {
	a.mu.RLock()
	h := a.factories[a.activeContext]
	mc := a.metricsClients[a.activeContext]
	a.mu.RUnlock()
	if h == nil {
		return
	}
	if h.IsForbidden("pods") {
		return
	}
	lister := h.Factory.Core().V1().Pods().Lister()

	allPods, err := kubeResources.ListPods(lister, nil)
	if err != nil {
		log.Printf("app: emitPods: %v", err)
		return
	}

	// Emit pods immediately without waiting for metrics
	if allMetrics != nil {
		allPods = kubeResources.ApplyPodMetrics(allPods, allMetrics)
	}
	runtime.EventsEmit(a.ctx, "pods:update", allPods)

	for _, ns := range namespaces {
		// Filter already-fetched cluster-wide data instead of re-listing
		nsPods := make([]dto.Pod, 0)
		for _, p := range allPods {
			if p.Namespace == ns {
				nsPods = append(nsPods, p)
			}
		}
		runtime.EventsEmit(a.ctx, "pods:"+ns+":update", nsPods)
	}

	// Fetch metrics asynchronously if needed to avoid blocking the emit
	if allMetrics == nil && mc != nil {
		go func() {
			ctx, cancel := context.WithTimeout(context.Background(), metricsFetchTimeout)
			defer cancel()
			fetchedMetrics := kube.FetchPodMetrics(ctx, mc, "")
			if fetchedMetrics != nil {
				// Apply metrics to the already-fetched pod list and re-emit; avoids a redundant relist.
				allPodsWithMetrics := kubeResources.ApplyPodMetrics(allPods, fetchedMetrics)
				runtime.EventsEmit(a.ctx, "pods:update", allPodsWithMetrics)

				for _, ns := range namespaces {
					// Filter to namespace and emit namespaced update with metrics
					nsPods := make([]dto.Pod, 0)
					for _, p := range allPodsWithMetrics {
						if p.Namespace == ns {
							nsPods = append(nsPods, p)
						}
					}
					runtime.EventsEmit(a.ctx, "pods:"+ns+":update", nsPods)
				}
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

	a.emitPods([]string{namespace})
	return nil
}

// DeletePods deletes multiple Pods, handling best-effort deletion across namespaces.
func (a *App) DeletePods(items []dto.PodRef) error {
	a.mu.RLock()
	cs := a.clients[a.activeContext]
	mc := a.metricsClients[a.activeContext]
	a.mu.RUnlock()
	if cs == nil {
		return fmt.Errorf("not connected")
	}

	var msgs []string
	namespaces := make(map[string]bool)

	for _, ref := range items {
		ctx, cancel := context.WithTimeout(context.Background(), apiMutationTimeout)
		err := cs.CoreV1().Pods(ref.Namespace).Delete(ctx, ref.Name, metav1.DeleteOptions{})
		cancel()
		if err != nil && !errors.IsNotFound(err) {
			msgs = append(msgs, fmt.Sprintf("%s/%s: %v", ref.Namespace, ref.Name, err))
		}
		namespaces[ref.Namespace] = true
	}

	// Fetch cluster-wide metrics once, then emit per-namespace using pre-fetched metrics
	var allMetrics map[string]dto.PodUsage
	if mc != nil {
		ctx, cancel := context.WithTimeout(context.Background(), metricsFetchTimeout)
		defer cancel()
		allMetrics = kube.FetchPodMetrics(ctx, mc, "")
	}

	// Emit updates for each unique namespace touched, using the pre-fetched metrics
	touchedNamespaces := make([]string, 0, len(namespaces))
	for ns := range namespaces {
		touchedNamespaces = append(touchedNamespaces, ns)
	}
	a.emitPodsWithMetrics(touchedNamespaces, allMetrics)

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

	a.emitPods([]string{namespace})

	return nil
}

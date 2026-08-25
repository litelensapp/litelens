package app

import (
	"log"
	"sort"

	kubeResources "github.com/litelensapp/litelens/internal/kube/resources"
	"github.com/litelensapp/litelens/packages/core/dto"
	"github.com/wailsapp/wails/v2/pkg/runtime"
)

func sortEventsDesc(events []dto.Event) {
	sort.Slice(events, func(i, j int) bool {
		return events[i].CreatedAt > events[j].CreatedAt
	})
}

func warningEvents(events []dto.Event) []dto.Event {
	sortEventsDesc(events)
	result := make([]dto.Event, 0, len(events))
	for _, e := range events {
		if e.Type == "Warning" {
			result = append(result, e)
		}
	}
	return result
}

func (a *App) ListEvents() ([]dto.Event, error) {
	a.mu.RLock()
	h := a.factories[a.activeContext]
	namespaces := a.activeNamespaces
	a.mu.RUnlock()
	if h == nil {
		return []dto.Event{}, nil
	}
	if h.IsForbidden("events") {
		return []dto.Event{}, nil
	}
	<-h.GetSyncedChan("events")
	if h.IsForbidden("events") {
		return nil, nil
	}
	result, err := kubeResources.ListEvents(
		h.Factory.Core().V1().Events().Lister(),
		namespaces,
	)
	if err != nil {
		log.Printf("app: ListEvents: %v", err)
		return []dto.Event{}, nil
	}
	return result, nil
}

func (a *App) ListWarningEvents() ([]dto.Event, error) {
	a.mu.RLock()
	h := a.factories[a.activeContext]
	namespaces := a.activeNamespaces
	a.mu.RUnlock()
	if h == nil {
		return []dto.Event{}, nil
	}
	if h.IsForbidden("events") {
		return []dto.Event{}, nil
	}
	<-h.GetSyncedChan("events")
	if h.IsForbidden("events") {
		return nil, nil
	}
	result, err := kubeResources.ListWarningEvents(
		h.Factory.Core().V1().Events().Lister(),
		namespaces,
	)
	if err != nil {
		log.Printf("app: ListWarningEvents: %v", err)
		return []dto.Event{}, nil
	}
	return result, nil
}

func (a *App) GetEventByName(namespace, name string) (dto.Event, error) {
	a.mu.RLock()
	h := a.factories[a.activeContext]
	a.mu.RUnlock()
	if h == nil {
		return dto.Event{}, nil
	}
	if h.IsForbidden("events") {
		return dto.Event{}, nil
	}
	<-h.GetSyncedChan("events")
	if h.IsForbidden("events") {
		return dto.Event{}, nil
	}
	result, err := kubeResources.GetEventByName(h.Factory.Core().V1().Events().Lister(), namespace, name)
	if err != nil {
		log.Printf("app: GetEventByName: %v", err)
		return dto.Event{}, nil
	}
	return result, nil
}

func (a *App) emitEvents() {
	a.mu.RLock()
	h := a.factories[a.activeContext]
	namespaces := a.activeNamespaces
	a.mu.RUnlock()
	if h == nil {
		return
	}
	if h.IsForbidden("events") {
		return
	}
	<-h.GetSyncedChan("events")
	if h.IsForbidden("events") {
		return
	}
	lister := h.Factory.Core().V1().Events().Lister()
	data, err := kubeResources.ListEvents(lister, namespaces)
	if err != nil {
		log.Printf("app: emitEvents: %v", err)
		return
	}
	sortEventsDesc(data)
	runtime.EventsEmit(a.ctx, "events:update", data)
	runtime.EventsEmit(a.ctx, "events:warning:update", warningEvents(data))
}

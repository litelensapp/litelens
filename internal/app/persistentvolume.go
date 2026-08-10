package app

import (
	"log"

	"github.com/gknguyen/litelens/internal/dto"
	"github.com/gknguyen/litelens/internal/kube/resources"
)

func (a *App) GetPersistentVolumeByName(name string) (dto.PersistentVolumeDetail, error) {
	a.mu.RLock()
	h := a.factories[a.activeContext]
	a.mu.RUnlock()
	if h == nil {
		return dto.PersistentVolumeDetail{}, nil
	}
	if h.IsForbidden("pvs") {
		return dto.PersistentVolumeDetail{}, nil
	}
	<-h.GetSyncedChan("pvs")
	if h.IsForbidden("pvs") {
		return dto.PersistentVolumeDetail{}, nil
	}
	result, err := kubeResources.GetPersistentVolumeByName(h.Factory.Core().V1().PersistentVolumes().Lister(), name)
	if err != nil {
		log.Printf("app: GetPersistentVolumeByName: %v", err)
		return dto.PersistentVolumeDetail{}, nil
	}
	return result, nil
}

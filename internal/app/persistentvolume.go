package app

import (
	"log"

	kubeResources "github.com/litelensapp/litelens/internal/kube/resources"
	"github.com/litelensapp/litelens/packages/core/kube/dto"
)

func (a *App) GetPersistentVolumeByName(name string) (dto.PersistentVolumeDetail, error) {
	h := a.activeFactory()
	if !waitForResourceSync(h, "pvs") {
		return dto.PersistentVolumeDetail{}, nil
	}
	result, err := kubeResources.GetPersistentVolumeByName(h.Factory.Core().V1().PersistentVolumes().Lister(), name)
	if err != nil {
		log.Printf("app: GetPersistentVolumeByName: %v", err)
		return dto.PersistentVolumeDetail{}, nil
	}
	return result, nil
}

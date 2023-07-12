package cluster

import (
	e "github.com/myntra/goscheduler/cluster_entity"
	"github.com/myntra/goscheduler/store"
)

type DummySupervisor struct {
}

// Implement if required
func (d *DummySupervisor) BootEntity(entity e.EntityInfo, bool bool) error {
	return nil
}

// Implement if required
func (d *DummySupervisor) DeactivateApp(app store.App) {
}

// Implement if required
func (d *DummySupervisor) ActivateApp(app store.App) {
}

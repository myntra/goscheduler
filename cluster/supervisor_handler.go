package cluster

import (
	e "github.com/myntra/goscheduler/cluster_entity"
	"github.com/myntra/goscheduler/store"
)

// SupervisorHandler is an interface for a supervisor's handler.
type SupervisorHandler interface {
	// BootEntity boots an entity with the specified entity info and reconcile flag.
	BootEntity(e.EntityInfo, bool) error
	// DeactivateApp deactivates the specified application.
	DeactivateApp(app store.App)
	// ActivateApp activates the specified application.
	ActivateApp(app store.App)
}

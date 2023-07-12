package cluster

import "github.com/myntra/goscheduler/cluster_entity"

// RecoverableEntity is a wrapper for a recoverable entity.
type RecoverableEntity struct {
	Obj       cluster_entity.Entity // The underlying entity.
	Recovered int                   // The number of times the entity was recovered.
}

// Stop stops the recoverable entity.
func (r RecoverableEntity) Stop() {
	r.Obj.Stop()
}

// Start starts the recoverable entity.
func (r RecoverableEntity) Start() {
	defer func() {
		// Recover from a panic and restart the entity if needed.
		if err := recover(); err != nil {
			r.Recovered++
			r.Obj.Init()
			r.Obj.Start()
		}
	}()

	// Initialize and start the entity.
	r.Obj.Init()
	r.Obj.Start()
}

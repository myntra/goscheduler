package cluster_entity

import (
	"github.com/myntra/goscheduler/retrieveriface"
)

// EntityFactory defines methods for creating and retrieving entities
type EntityFactory interface {
	CreateEntity(id string) Entity                            // CreateEntity creates a new Entity with the given ID
	GetEntityRetriever(appID string) retrieveriface.Retriever // GetEntityRetriever returns a retriever for the specified app ID
}

// Entity represents a cluster entity with methods for starting, stopping, and initializing it
type Entity interface {
	Start()      // Start starts the entity
	Stop()       // Stop stops the entity
	Init() error // Init initializes the entity and returns an error if there is any
}

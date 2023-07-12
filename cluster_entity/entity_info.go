package cluster_entity

import (
	"strconv"
	"strings"
)

// EntityInfo represents a cluster entity and its properties
type EntityInfo struct {
	Id      string // ID of the entity
	Node    string // Node that the entity is running on
	Status  int    // Status code for the entity
	History string // History of the entity
}

// GetAppName returns the name of the application that the entity belongs to
func (entity EntityInfo) GetAppName() string {
	appName := entity.Id[:strings.LastIndex(entity.Id, ".")]
	return appName
}

// GetPartitionId returns the partition ID of the entity
func (entity EntityInfo) GetPartitionId() int {
	partitionId, _ := strconv.Atoi(entity.Id[strings.LastIndex(entity.Id, ".")+1:])
	return partitionId
}

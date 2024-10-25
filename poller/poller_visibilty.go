package poller

import (
	"sync"
	"time"
)

// PollerMetrics represents metrics for a single poller
type PollerMetrics struct {
	ID            string    `json:"id"`
	AppName       string    `json:"appName"`
	PartitionID   int       `json:"partitionId"`
	Node          string    `json:"node"`
	Status        string    `json:"status"`
	LastActive    time.Time `json:"lastActive"`
	JobsExecuted  int64     `json:"jobsExecuted"`
	JobsSucceeded int64     `json:"jobsSucceeded"`
	JobsFailed    int64     `json:"jobsFailed"`
	LastError     string    `json:"lastError"`
	CreatedAt     time.Time `json:"createdAt"`
}

// AppPollerMetrics represents metrics for all pollers of an app
type AppPollerMetrics struct {
	AppName           string          `json:"appName"`
	TotalPollers      int             `json:"totalPollers"`
	ActivePollers     int             `json:"activePollers"`
	NodeDistribution  map[string]int  `json:"nodeDistribution"`
	PollerMetrics     []PollerMetrics `json:"pollerMetrics"`
	TotalJobsExecuted int64           `json:"totalJobsExecuted"`
	TotalJobsFailed   int64           `json:"totalJobsFailed"`
	UpdatedAt         time.Time       `json:"updatedAt"`
}

// Global metrics store instance
var (
	globalMetricsStore *MetricsStore
	once               sync.Once
)

// GetMetricsStore returns the singleton instance of MetricsStore
func GetMetricsStore() *MetricsStore {
	once.Do(func() {
		globalMetricsStore = &MetricsStore{
			metricsCache: make(map[string]AppPollerMetrics),
		}
	})
	return globalMetricsStore
}

// MetricsStore handles storage and retrieval of poller metrics
type MetricsStore struct {
	mu           sync.RWMutex
	metricsCache map[string]AppPollerMetrics // key: appName
}

// NewMetricsStore creates a new metrics store
func NewMetricsStore() *MetricsStore {
	return &MetricsStore{
		metricsCache: make(map[string]AppPollerMetrics),
	}
}

// UpdatePollerMetrics updates metrics for a single poller
func (ms *MetricsStore) UpdatePollerMetrics(metrics PollerMetrics) {
	ms.mu.Lock()
	defer ms.mu.Unlock()

	appMetrics, exists := ms.metricsCache[metrics.AppName]
	if !exists {
		appMetrics = AppPollerMetrics{
			AppName:          metrics.AppName,
			NodeDistribution: make(map[string]int),
			PollerMetrics:    make([]PollerMetrics, 0),
		}
	}

	// Update or add poller metrics
	found := false
	for i, pm := range appMetrics.PollerMetrics {
		if pm.ID == metrics.ID {
			appMetrics.PollerMetrics[i] = metrics
			found = true
			break
		}
	}
	if !found {
		appMetrics.PollerMetrics = append(appMetrics.PollerMetrics, metrics)
	}

	// Recalculate app-level metrics
	appMetrics.UpdatedAt = time.Now()
	appMetrics.TotalPollers = len(appMetrics.PollerMetrics)
	appMetrics.ActivePollers = 0
	appMetrics.TotalJobsExecuted = 0
	appMetrics.TotalJobsFailed = 0
	appMetrics.NodeDistribution = make(map[string]int)

	for _, pm := range appMetrics.PollerMetrics {
		if pm.Status == "active" {
			appMetrics.ActivePollers++
		}
		appMetrics.TotalJobsExecuted += pm.JobsExecuted
		appMetrics.TotalJobsFailed += pm.JobsFailed
		appMetrics.NodeDistribution[pm.Node]++
	}

	ms.metricsCache[metrics.AppName] = appMetrics
}

// GetAppMetrics retrieves metrics for a specific app
func (ms *MetricsStore) GetAppMetrics(appName string) (AppPollerMetrics, bool) {
	ms.mu.RLock()
	defer ms.mu.RUnlock()
	metrics, exists := ms.metricsCache[appName]
	return metrics, exists
}

// GetAllMetrics retrieves metrics for all apps
func (ms *MetricsStore) GetAllMetrics() map[string]AppPollerMetrics {
	ms.mu.RLock()
	defer ms.mu.RUnlock()

	// Create a copy to avoid concurrent access issues
	result := make(map[string]AppPollerMetrics)
	for k, v := range ms.metricsCache {
		result[k] = v
	}
	return result
}

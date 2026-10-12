// Copyright (c) 2023 Myntra Designs Private Limited.
//
// Permission is hereby granted, free of charge, to any person obtaining a copy of
// this software and associated documentation files (the "Software"), to deal in
// the Software without restriction, including without limitation the rights to
// use, copy, modify, merge, publish, distribute, sublicense, and/or sell copies of
// the Software, and to permit persons to whom the Software is furnished to do so,
// subject to the following conditions:
//
// The above copyright notice and this permission notice shall be included in all
// copies or substantial portions of the Software.
//
// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY, FITNESS
// FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE AUTHORS OR
// COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER LIABILITY, WHETHER
// IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM, OUT OF OR IN
// CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE SOFTWARE.

package cluster

import (
	"errors"
	"fmt"
	"sort"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/myntra/goscheduler/cluster_entity"
	"github.com/myntra/goscheduler/dao"
	"github.com/myntra/goscheduler/retrieveriface"
	"github.com/myntra/goscheduler/store"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/uber/ringpop-go/events"
)

const (
	selfNode  = "10.0.0.1:9091"
	nodeB     = "10.0.0.2:9091"
	nodeC     = "10.0.0.3:9091"
	waitLimit = 2 * time.Second
)

// fakeRing maps entity IDs to owners; IDs without an owner fail lookup.
type fakeRing struct {
	mu     sync.Mutex
	owners map[string]string
}

func newFakeRing(owners map[string]string) *fakeRing {
	return &fakeRing{owners: owners}
}

func (r *fakeRing) set(id, owner string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.owners[id] = owner
}

func (r *fakeRing) ownerOf(id string) (string, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	owner, ok := r.owners[id]
	if !ok {
		return "", fmt.Errorf("no owner for %s", id)
	}
	return owner, nil
}

type countingEntity struct {
	inits  int32
	starts int32
	stops  int32
}

func (c *countingEntity) Init() error { atomic.AddInt32(&c.inits, 1); return nil }
func (c *countingEntity) Start()      { atomic.AddInt32(&c.starts, 1) }
func (c *countingEntity) Stop()       { atomic.AddInt32(&c.stops, 1) }

func (c *countingEntity) stopCount() int32 { return atomic.LoadInt32(&c.stops) }
func (c *countingEntity) initCount() int32 { return atomic.LoadInt32(&c.inits) }

type bulkAction struct {
	app         string
	partitionId int
	actionType  store.ActionType
}

type recordingRetriever struct {
	mu      sync.Mutex
	actions []bulkAction
}

func (r *recordingRetriever) GetSchedules(string, int, time.Time) error { return nil }

func (r *recordingRetriever) BulkAction(app store.App, partitionId int, _ time.Time, _ []store.Status, actionType store.ActionType) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.actions = append(r.actions, bulkAction{app: app.AppId, partitionId: partitionId, actionType: actionType})
	return nil
}

func (r *recordingRetriever) recorded() []bulkAction {
	r.mu.Lock()
	defer r.mu.Unlock()
	return append([]bulkAction(nil), r.actions...)
}

type recordingFactory struct {
	mu        sync.Mutex
	created   map[string]*countingEntity
	creations int
	retriever *recordingRetriever
	// createDelay widens the gap between checking for an entity and storing it,
	// so concurrent starts that are not serialized reliably create duplicates.
	createDelay time.Duration
}

func newRecordingFactory() *recordingFactory {
	return &recordingFactory{created: map[string]*countingEntity{}, retriever: &recordingRetriever{}}
}

func (f *recordingFactory) CreateEntity(id string) cluster_entity.Entity {
	time.Sleep(f.createDelay)
	f.mu.Lock()
	defer f.mu.Unlock()
	entity := &countingEntity{}
	f.created[id] = entity
	f.creations++
	return entity
}

func (f *recordingFactory) GetEntityRetriever(string) retrieveriface.Retriever { return f.retriever }

func (f *recordingFactory) createdEntity(id string) (*countingEntity, bool) {
	f.mu.Lock()
	defer f.mu.Unlock()
	entity, ok := f.created[id]
	return entity, ok
}

type statusUpdate struct {
	id     string
	node   string
	status int
}

// recordingClusterDao records entity status writes; apps default to active.
type recordingClusterDao struct {
	dao.DummyClusterDaoImpl
	mu        sync.Mutex
	updates   []statusUpdate
	updateErr error
	apps      map[string]store.App
	appErr    error
}

func (d *recordingClusterDao) UpdateEntityStatus(id string, nodeName string, status int) error {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.updates = append(d.updates, statusUpdate{id: id, node: nodeName, status: status})
	return d.updateErr
}

func (d *recordingClusterDao) GetApp(appName string) (store.App, error) {
	d.mu.Lock()
	defer d.mu.Unlock()
	if d.appErr != nil {
		return store.App{}, d.appErr
	}
	if app, ok := d.apps[appName]; ok {
		return app, nil
	}
	return store.App{AppId: appName, Partitions: 1, Active: true}, nil
}

func (d *recordingClusterDao) statusUpdates() []statusUpdate {
	d.mu.Lock()
	defer d.mu.Unlock()
	return append([]statusUpdate(nil), d.updates...)
}

type handoffCall struct {
	dest string
	ids  []string
}

// fakeHandoff records handoff requests and answers them with respond.
type fakeHandoff struct {
	mu      sync.Mutex
	calls   []handoffCall
	respond func(dest string, ids []string) ([]string, error)
}

func acceptAll(_ string, ids []string) ([]string, error) { return ids, nil }

func (h *fakeHandoff) handoff(dest string, ids []string) ([]string, error) {
	sorted := append([]string(nil), ids...)
	sort.Strings(sorted)
	h.mu.Lock()
	h.calls = append(h.calls, handoffCall{dest: dest, ids: sorted})
	respond := h.respond
	h.mu.Unlock()
	return respond(dest, ids)
}

func (h *fakeHandoff) recorded() []handoffCall {
	h.mu.Lock()
	defer h.mu.Unlock()
	calls := append([]handoffCall(nil), h.calls...)
	sort.Slice(calls, func(i, j int) bool { return calls[i].dest < calls[j].dest })
	return calls
}

func (h *fakeHandoff) callCount() int {
	h.mu.Lock()
	defer h.mu.Unlock()
	return len(h.calls)
}

type testNode struct {
	s       *Supervisor
	ring    *fakeRing
	dao     *recordingClusterDao
	factory *recordingFactory
	handoff *fakeHandoff
}

func newTestNode(t *testing.T, owners map[string]string, opts ...Option) *testNode {
	t.Helper()
	node := &testNode{
		ring:    newFakeRing(owners),
		dao:     &recordingClusterDao{apps: map[string]store.App{}},
		factory: newRecordingFactory(),
		handoff: &fakeHandoff{respond: acceptAll},
	}
	defaults := []Option{
		WithReconciliationEnabled(true),
		WithReconciliationOffset(2),
		WithRebalanceInterval(0),
		WithRebalanceSettleDelay(0),
	}
	node.s = NewSupervisor(node.factory, node.dao, nil, append(defaults, opts...)...)
	node.s.address = selfNode
	node.s.ownerOf = node.ring.ownerOf
	node.s.handoff = node.handoff.handoff
	t.Cleanup(node.s.stopRebalancer)
	return node
}

// running registers an already running entity, as if started earlier.
func (n *testNode) running(id string) *countingEntity {
	entity := &countingEntity{}
	n.s.entities.Set(id, RecoverableEntity{Obj: entity})
	return entity
}

func (n *testNode) isRunning(id string) bool {
	_, ok := n.s.entities.Get(id)
	return ok
}

func eventually(t *testing.T, condition func() bool, msg string) {
	t.Helper()
	deadline := time.Now().Add(waitLimit)
	for time.Now().Before(deadline) {
		if condition() {
			return
		}
		time.Sleep(5 * time.Millisecond)
	}
	t.Fatal(msg)
}

func takeOver(t *testing.T, n *testNode, ids ...string) *Response {
	t.Helper()
	res, err := n.s.TakeOverEntities(nil, &EntityIDs{Ids: ids})
	require.NoError(t, err)
	assert.Equal(t, SUCCESS, res.Status)
	assert.Equal(t, selfNode, res.ServerAddress)
	return res
}

func TestRebalanceHandsOffOnlyEntitiesOwnedElsewhere(t *testing.T) {
	n := newTestNode(t, map[string]string{
		"own.0":   selfNode,
		"b.0":     nodeB,
		"b.1":     nodeB,
		"c.0":     nodeC,
		"own.1":   selfNode,
		"other.4": nodeB,
	})
	own0, own1 := n.running("own.0"), n.running("own.1")
	b0, b1, c0, other4 := n.running("b.0"), n.running("b.1"), n.running("c.0"), n.running("other.4")

	n.s.rebalanceEntities()

	assert.Equal(t, []handoffCall{
		{dest: nodeB, ids: []string{"b.0", "b.1", "other.4"}},
		{dest: nodeC, ids: []string{"c.0"}},
	}, n.handoff.recorded())
	for id, entity := range map[string]*countingEntity{"b.0": b0, "b.1": b1, "c.0": c0, "other.4": other4} {
		assert.False(t, n.isRunning(id), "%s should be released", id)
		assert.Equal(t, int32(1), entity.stopCount(), "%s should be stopped once", id)
	}
	for id, entity := range map[string]*countingEntity{"own.0": own0, "own.1": own1} {
		assert.True(t, n.isRunning(id), "%s is owned here and must keep running", id)
		assert.Equal(t, int32(0), entity.stopCount())
	}
	assert.Empty(t, n.dao.statusUpdates(), "releasing must not overwrite the owner's DB record")
}

func TestRebalanceKeepsEntitiesWhenHandoffFails(t *testing.T) {
	n := newTestNode(t, map[string]string{"b.0": nodeB, "b.1": nodeB})
	b0, b1 := n.running("b.0"), n.running("b.1")
	n.handoff.respond = func(string, []string) ([]string, error) {
		return nil, errors.New("no handler for TakeOverEntities")
	}

	n.s.rebalanceEntities()

	assert.Equal(t, 1, n.handoff.callCount())
	assert.True(t, n.isRunning("b.0"))
	assert.True(t, n.isRunning("b.1"))
	assert.Equal(t, int32(0), b0.stopCount())
	assert.Equal(t, int32(0), b1.stopCount())
}

func TestRebalanceReleasesOnlyEntitiesTheOwnerAccepted(t *testing.T) {
	n := newTestNode(t, map[string]string{"own.0": selfNode, "b.0": nodeB, "b.1": nodeB})
	n.running("own.0")
	n.running("b.0")
	b1 := n.running("b.1")
	// A response must never release entities that were not part of the request.
	n.handoff.respond = func(string, []string) ([]string, error) {
		return []string{"b.0", "own.0", "unknown.7"}, nil
	}

	n.s.rebalanceEntities()

	assert.False(t, n.isRunning("b.0"))
	assert.True(t, n.isRunning("b.1"), "rejected entity must keep running")
	assert.Equal(t, int32(0), b1.stopCount())
	assert.True(t, n.isRunning("own.0"), "entity not in the request must not be released")
}

func TestRebalanceKeepsEntityWhoseOwnerChangedDuringHandoff(t *testing.T) {
	for name, newOwner := range map[string]string{
		"moved back to this node": selfNode,
		"moved to a third node":   nodeC,
	} {
		t.Run(name, func(t *testing.T) {
			n := newTestNode(t, map[string]string{"b.0": nodeB})
			entity := n.running("b.0")
			n.handoff.respond = func(_ string, ids []string) ([]string, error) {
				n.ring.set("b.0", newOwner)
				return ids, nil
			}

			n.s.rebalanceEntities()

			assert.True(t, n.isRunning("b.0"))
			assert.Equal(t, int32(0), entity.stopCount())
		})
	}
}

func TestRebalanceKeepsEntitiesWhoseOwnerCannotBeLookedUp(t *testing.T) {
	n := newTestNode(t, map[string]string{"b.0": nodeB})
	n.running("b.0")
	orphan := n.running("unknown.0")

	n.s.rebalanceEntities()

	assert.Equal(t, []handoffCall{{dest: nodeB, ids: []string{"b.0"}}}, n.handoff.recorded())
	assert.True(t, n.isRunning("unknown.0"))
	assert.Equal(t, int32(0), orphan.stopCount())
}

func TestRebalanceDoesNothingWhenAllEntitiesAreOwnedHere(t *testing.T) {
	n := newTestNode(t, map[string]string{"own.0": selfNode, "own.1": selfNode})
	n.running("own.0")
	n.running("own.1")

	n.s.rebalanceEntities()

	assert.Equal(t, 0, n.handoff.callCount())
	assert.Equal(t, 2, n.s.entities.Count())
}

func TestRebalanceRecoversFromPanickingHandoff(t *testing.T) {
	n := newTestNode(t, map[string]string{"b.0": nodeB})
	n.running("b.0")
	n.handoff.respond = func(string, []string) ([]string, error) { panic("forward exploded") }

	assert.NotPanics(t, n.s.rebalanceEntities)
	assert.True(t, n.isRunning("b.0"))
}

func TestRebalanceStopsHandingOffOnceStopped(t *testing.T) {
	n := newTestNode(t, map[string]string{"b.0": nodeB, "c.0": nodeC})
	n.running("b.0")
	n.running("c.0")
	n.s.stopRebalancer()

	n.s.rebalanceEntities()

	assert.Equal(t, 0, n.handoff.callCount())
	assert.Equal(t, 2, n.s.entities.Count())
}

func TestTakeOverStartsOwnedEntityAndReconcilesRecentSchedules(t *testing.T) {
	n := newTestNode(t, map[string]string{"app.3": selfNode})

	res := takeOver(t, n, "app.3")

	assert.Equal(t, []string{"app.3"}, res.Accepted)
	assert.True(t, n.isRunning("app.3"))
	created, ok := n.factory.createdEntity("app.3")
	require.True(t, ok, "entity should be created by the factory")
	eventually(t, func() bool { return created.initCount() == 1 }, "entity was never started")
	assert.Equal(t, []statusUpdate{{id: "app.3", node: selfNode, status: RUNNING}}, n.dao.statusUpdates())
	// Offset 2 reconciles the current minute and the two before it.
	assert.Equal(t, []bulkAction{
		{app: "app", partitionId: 3, actionType: store.Reconcile},
		{app: "app", partitionId: 3, actionType: store.Reconcile},
		{app: "app", partitionId: 3, actionType: store.Reconcile},
	}, n.factory.retriever.recorded())
}

func TestTakeOverOfRunningEntityReassertsOwnershipWithoutRestart(t *testing.T) {
	n := newTestNode(t, map[string]string{"app.3": selfNode})
	entity := n.running("app.3")

	res := takeOver(t, n, "app.3")

	assert.Equal(t, []string{"app.3"}, res.Accepted)
	assert.Equal(t, []statusUpdate{{id: "app.3", node: selfNode, status: RUNNING}}, n.dao.statusUpdates())
	_, created := n.factory.createdEntity("app.3")
	assert.False(t, created, "running entity must not be recreated")
	time.Sleep(20 * time.Millisecond)
	assert.Equal(t, int32(0), entity.initCount(), "running entity must not be restarted")
	assert.Equal(t, int32(0), entity.stopCount())
	assert.Empty(t, n.factory.retriever.recorded(), "no reconciliation for an entity that never stopped")
}

func TestTakeOverRejectsEntitiesThisNodeDoesNotOwn(t *testing.T) {
	n := newTestNode(t, map[string]string{"b.0": nodeB})

	res := takeOver(t, n, "b.0", "unknown.0")

	assert.Empty(t, res.Accepted)
	assert.Equal(t, 0, n.s.entities.Count())
	assert.Empty(t, n.dao.statusUpdates())
	assert.Empty(t, n.factory.retriever.recorded())
}

func TestTakeOverRejectsWhenAppCannotBeLoaded(t *testing.T) {
	n := newTestNode(t, map[string]string{"app.0": selfNode})
	n.dao.appErr = errors.New("cassandra timeout")

	res := takeOver(t, n, "app.0")

	assert.Empty(t, res.Accepted)
	assert.False(t, n.isRunning("app.0"))
}

func TestTakeOverAcceptsEntityOfInactiveAppWithoutStartingIt(t *testing.T) {
	n := newTestNode(t, map[string]string{"paused.0": selfNode})
	n.dao.apps["paused"] = store.App{AppId: "paused", Partitions: 1, Active: false}

	res := takeOver(t, n, "paused.0")

	assert.Equal(t, []string{"paused.0"}, res.Accepted, "the sender should stop polling a deactivated app")
	assert.False(t, n.isRunning("paused.0"))
	assert.Empty(t, n.dao.statusUpdates())
	assert.Empty(t, n.factory.retriever.recorded())
}

func TestTakeOverRejectsRunningEntityWhenStatusUpdateFails(t *testing.T) {
	n := newTestNode(t, map[string]string{"app.0": selfNode})
	n.running("app.0")
	n.dao.updateErr = errors.New("write timeout")

	res := takeOver(t, n, "app.0")

	assert.Empty(t, res.Accepted, "sender must keep its copy until ownership is recorded")
	assert.True(t, n.isRunning("app.0"))
}

func TestTakeOverHandlesEachEntityIndependently(t *testing.T) {
	n := newTestNode(t, map[string]string{
		"good.0":       selfNode,
		"noseparator":  selfNode,
		"elsewhere.0":  nodeB,
		"good.1":       selfNode,
		"alreadyrun.2": selfNode,
	})
	n.running("alreadyrun.2")

	res := takeOver(t, n, "good.0", "noseparator", "elsewhere.0", "good.1", "alreadyrun.2")

	assert.Equal(t, []string{"good.0", "good.1", "alreadyrun.2"}, res.Accepted)
	assert.False(t, n.isRunning("noseparator"), "malformed id must be rejected, not crash the handler")
	assert.False(t, n.isRunning("elsewhere.0"))
}

func TestTakeOverSkipsReconciliationWhenDisabled(t *testing.T) {
	n := newTestNode(t, map[string]string{"app.0": selfNode}, WithReconciliationEnabled(false))

	res := takeOver(t, n, "app.0")

	assert.Equal(t, []string{"app.0"}, res.Accepted)
	assert.True(t, n.isRunning("app.0"))
	assert.Empty(t, n.factory.retriever.recorded())
}

func TestConcurrentTakeOversOfOneEntityStartItOnce(t *testing.T) {
	n := newTestNode(t, map[string]string{"app.0": selfNode}, WithReconciliationEnabled(false))
	n.factory.createDelay = 20 * time.Millisecond

	var wg sync.WaitGroup
	for i := 0; i < 20; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			res, err := n.s.TakeOverEntities(nil, &EntityIDs{Ids: []string{"app.0"}})
			assert.NoError(t, err)
			assert.Equal(t, []string{"app.0"}, res.Accepted)
		}()
	}
	wg.Wait()

	assert.Equal(t, 1, n.factory.createdCount(), "the entity must be created exactly once")
	updates := n.dao.statusUpdates()
	assert.Len(t, updates, 20)
	for _, update := range updates {
		assert.Equal(t, statusUpdate{id: "app.0", node: selfNode, status: RUNNING}, update)
	}
}

func (f *recordingFactory) createdCount() int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.creations
}

func TestConcurrentStartEntityAndTakeOverCreateOneEntity(t *testing.T) {
	for i := 0; i < 5; i++ {
		n := newTestNode(t, map[string]string{"app.0": selfNode}, WithReconciliationEnabled(false))
		n.factory.createDelay = 20 * time.Millisecond
		start := make(chan struct{})
		var wg sync.WaitGroup
		wg.Add(2)
		go func() {
			defer wg.Done()
			<-start
			_, err := n.s.StartEntity("app.0")
			assert.NoError(t, err)
		}()
		go func() {
			defer wg.Done()
			<-start
			res, err := n.s.TakeOverEntities(nil, &EntityIDs{Ids: []string{"app.0"}})
			assert.NoError(t, err)
			assert.Equal(t, []string{"app.0"}, res.Accepted)
		}()
		close(start)
		wg.Wait()

		require.Equal(t, 1, n.factory.createdCount(), "a second poller would be orphaned with no way to stop it")
	}
}

func TestConcurrentStartEntityCallsCreateOneEntity(t *testing.T) {
	n := newTestNode(t, map[string]string{})
	n.factory.createDelay = 20 * time.Millisecond
	var wg sync.WaitGroup
	for i := 0; i < 20; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, err := n.s.StartEntity("app.0")
			assert.NoError(t, err)
		}()
	}
	wg.Wait()

	assert.Equal(t, 1, n.factory.createdCount())
	assert.Equal(t, 1, n.s.entities.Count())
}

func TestStopEntityRecordsStopButReleaseDoesNot(t *testing.T) {
	n := newTestNode(t, map[string]string{"stop.0": selfNode, "release.0": nodeB})
	stopped := n.running("stop.0")
	released := n.running("release.0")

	exists, err := n.s.StopEntity("stop.0")
	require.NoError(t, err)
	assert.True(t, exists)
	assert.True(t, n.s.releaseEntity("release.0", nodeB))

	assert.Equal(t, int32(1), stopped.stopCount())
	assert.Equal(t, int32(1), released.stopCount())
	assert.Equal(t, 0, n.s.entities.Count())
	assert.Equal(t, []statusUpdate{{id: "stop.0", node: selfNode, status: STOPPED}}, n.dao.statusUpdates())
}

func TestStopEntityOfUnknownEntityIsNoop(t *testing.T) {
	n := newTestNode(t, map[string]string{})

	exists, err := n.s.StopEntity("missing.0")

	require.NoError(t, err)
	assert.False(t, exists)
	assert.Empty(t, n.dao.statusUpdates())
}

func TestEnsureEntityRunningKeepsRunningEntity(t *testing.T) {
	n := newTestNode(t, map[string]string{})
	entity := n.running("app.0")

	started, err := n.s.ensureEntityRunning("app.0")

	require.NoError(t, err)
	assert.False(t, started)
	time.Sleep(20 * time.Millisecond)
	assert.Equal(t, int32(0), entity.initCount())
	assert.Equal(t, []statusUpdate{{id: "app.0", node: selfNode, status: RUNNING}}, n.dao.statusUpdates())
}

func TestDisabledRebalancerNeverHandsOffButStillAcceptsTakeovers(t *testing.T) {
	n := newTestNode(t, map[string]string{"b.0": nodeB, "own.0": selfNode},
		WithRebalanceEnabled(false), WithRebalanceInterval(10*time.Millisecond))
	n.running("b.0")
	n.s.startRebalancer()

	n.s.HandleEvent(events.RingChangedEvent{ServersAdded: []string{nodeB}})
	time.Sleep(50 * time.Millisecond)

	assert.Equal(t, 0, n.handoff.callCount())
	assert.True(t, n.isRunning("b.0"))
	assert.Equal(t, []string{"own.0"}, takeOver(t, n, "own.0").Accepted,
		"a node with rebalancing off must still accept entities from peers that have it on")
	assert.NotPanics(t, n.s.stopRebalancer)
}

func TestWithRebalance(t *testing.T) {
	for name, tc := range map[string]struct {
		enabled         bool
		intervalSeconds int
		settleSeconds   int
		want            options
	}{
		"configured values":       {true, 30, 2, options{rebalanceEnabled: true, rebalanceInterval: 30 * time.Second, rebalanceSettleDelay: 2 * time.Second}},
		"zero keeps defaults":     {true, 0, 0, options{rebalanceEnabled: true, rebalanceInterval: time.Minute, rebalanceSettleDelay: 5 * time.Second}},
		"negative keeps defaults": {true, -1, -1, options{rebalanceEnabled: true, rebalanceInterval: time.Minute, rebalanceSettleDelay: 5 * time.Second}},
		"disabled":                {false, 60, 5, options{rebalanceEnabled: false, rebalanceInterval: time.Minute, rebalanceSettleDelay: 5 * time.Second}},
	} {
		t.Run(name, func(t *testing.T) {
			s := NewSupervisor(newRecordingFactory(), &recordingClusterDao{}, nil,
				WithRebalance(tc.enabled, tc.intervalSeconds, tc.settleSeconds))

			assert.Equal(t, tc.want.rebalanceEnabled, s.opt.rebalanceEnabled)
			assert.Equal(t, tc.want.rebalanceInterval, s.opt.rebalanceInterval)
			assert.Equal(t, tc.want.rebalanceSettleDelay, s.opt.rebalanceSettleDelay)
		})
	}
}

func TestRebalanceIsEnabledByDefault(t *testing.T) {
	s := NewSupervisor(newRecordingFactory(), &recordingClusterDao{}, nil)

	assert.True(t, s.opt.rebalanceEnabled)
	assert.Equal(t, time.Minute, s.opt.rebalanceInterval)
	assert.Equal(t, 5*time.Second, s.opt.rebalanceSettleDelay)
}

func TestRingChangeTriggersRebalance(t *testing.T) {
	for name, event := range map[string]events.RingChangedEvent{
		"node added":   {ServersAdded: []string{nodeB}},
		"node removed": {ServersRemoved: []string{nodeC}},
		"node updated": {ServersUpdated: []string{nodeB}},
	} {
		t.Run(name, func(t *testing.T) {
			n := newTestNode(t, map[string]string{"b.0": nodeB}, WithRebalanceSettleDelay(10*time.Millisecond))
			n.running("b.0")
			n.s.startRebalancer()

			n.s.HandleEvent(event)

			eventually(t, func() bool { return !n.isRunning("b.0") }, "entity was not handed back after a ring change")
			assert.Equal(t, []handoffCall{{dest: nodeB, ids: []string{"b.0"}}}, n.handoff.recorded())
		})
	}
}

func TestOtherRingEventsDoNotTriggerRebalance(t *testing.T) {
	n := newTestNode(t, map[string]string{"b.0": nodeB})
	n.running("b.0")
	n.s.startRebalancer()

	n.s.HandleEvent(events.RingChecksumEvent{})

	time.Sleep(50 * time.Millisecond)
	assert.Equal(t, 0, n.handoff.callCount())
	assert.True(t, n.isRunning("b.0"))
}

func TestBurstOfRingChangesIsDebouncedIntoOneRebalance(t *testing.T) {
	n := newTestNode(t, map[string]string{"b.0": nodeB}, WithRebalanceSettleDelay(150*time.Millisecond))
	n.running("b.0")
	// Keep the entity misplaced so every sweep would issue a handoff.
	n.handoff.respond = func(string, []string) ([]string, error) { return nil, errors.New("rejected") }
	n.s.startRebalancer()

	for i := 0; i < 50; i++ {
		n.s.HandleEvent(events.RingChangedEvent{ServersAdded: []string{nodeB}})
	}

	eventually(t, func() bool { return n.handoff.callCount() == 1 }, "expected one rebalance")
	time.Sleep(300 * time.Millisecond)
	assert.Equal(t, 1, n.handoff.callCount(), "a flapping ring must not cause a sweep per event")
}

func TestPeriodicSweepRetriesFailedHandoffs(t *testing.T) {
	n := newTestNode(t, map[string]string{"b.0": nodeB}, WithRebalanceInterval(20*time.Millisecond))
	entity := n.running("b.0")
	var attempts int32
	n.handoff.respond = func(_ string, ids []string) ([]string, error) {
		if atomic.AddInt32(&attempts, 1) < 3 {
			return nil, errors.New("owner unreachable")
		}
		return ids, nil
	}
	n.s.startRebalancer()

	eventually(t, func() bool { return !n.isRunning("b.0") }, "periodic sweep never handed the entity back")
	assert.GreaterOrEqual(t, atomic.LoadInt32(&attempts), int32(3))
	assert.Equal(t, int32(1), entity.stopCount())
}

func TestStoppedRebalancerIgnoresRingChanges(t *testing.T) {
	n := newTestNode(t, map[string]string{"b.0": nodeB})
	n.running("b.0")
	n.s.startRebalancer()

	n.s.stopRebalancer()
	n.s.stopRebalancer()
	select {
	case <-n.s.rebalanceDone:
	case <-time.After(waitLimit):
		t.Fatal("rebalancer did not exit")
	}
	n.s.HandleEvent(events.RingChangedEvent{ServersAdded: []string{nodeB}})
	n.s.startRebalancer()

	time.Sleep(50 * time.Millisecond)
	assert.Equal(t, 0, n.handoff.callCount())
	assert.True(t, n.isRunning("b.0"))
}

func TestStopRebalancerWithoutStartDoesNotBlock(t *testing.T) {
	n := newTestNode(t, map[string]string{})
	done := make(chan struct{})
	go func() {
		n.s.stopRebalancer()
		close(done)
	}()
	select {
	case <-done:
	case <-time.After(waitLimit):
		t.Fatal("stopRebalancer blocked when the rebalancer never started")
	}
}

func TestStopRebalancerInterruptsSettleDelay(t *testing.T) {
	n := newTestNode(t, map[string]string{"b.0": nodeB}, WithRebalanceSettleDelay(time.Hour))
	n.running("b.0")
	n.s.startRebalancer()
	n.s.requestRebalance()

	n.s.stopRebalancer()

	select {
	case <-n.s.rebalanceDone:
	case <-time.After(waitLimit):
		t.Fatal("rebalancer kept waiting out the settle delay after stop")
	}
	assert.Equal(t, 0, n.handoff.callCount())
}

func TestStopNodeStopsRebalancer(t *testing.T) {
	n := newTestNode(t, map[string]string{"own.0": selfNode})
	entity := n.running("own.0")
	n.s.startRebalancer()

	n.s.StopNode()

	select {
	case <-n.s.rebalanceDone:
	case <-time.After(waitLimit):
		t.Fatal("StopNode left the rebalancer running")
	}
	assert.Equal(t, 0, n.s.entities.Count())
	assert.Equal(t, int32(1), entity.stopCount())
}

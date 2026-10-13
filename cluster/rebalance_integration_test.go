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
	"fmt"
	"testing"
	"time"

	e "github.com/myntra/goscheduler/cluster_entity"
	"github.com/myntra/goscheduler/store"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type ringpopNode struct {
	s       *Supervisor
	dao     *recordingClusterDao
	factory *recordingFactory
}

func (n *ringpopNode) running(id string) *countingEntity {
	entity := &countingEntity{}
	n.s.entities.Set(id, RecoverableEntity{Obj: entity})
	return entity
}

func (n *ringpopNode) isRunning(id string) bool {
	_, ok := n.s.entities.Get(id)
	return ok
}

// startTwoNodeCluster joins two supervisors on loopback. Automatic sweeps are
// disabled so each test decides exactly when a node rebalances.
func startTwoNodeCluster(t *testing.T) (*ringpopNode, *ringpopNode) {
	t.Helper()
	const addr1, addr2 = "127.0.0.1:2383", "127.0.0.1:2384"

	newNode := func(address string, seeds ...string) *ringpopNode {
		n := &ringpopNode{
			dao:     &recordingClusterDao{apps: map[string]store.App{}},
			factory: newRecordingFactory(),
		}
		n.s = NewSupervisor(n.factory, n.dao, nil,
			WithClusterName("rebalance-test"),
			WithAddress(address),
			WithBootStrapServers(seeds),
			WithJoinSize(1),
			WithLogEnabled(false),
			WithReplicaPoints(100),
			WithReconciliationEnabled(true),
			WithReconciliationOffset(1),
			WithRebalanceInterval(0),
			WithRebalanceSettleDelay(time.Hour))
		return n
	}

	n1, n2 := newNode(addr1, addr1), newNode(addr2, addr1, addr2)
	n1.s.InitRingPop()
	n2.s.InitRingPop()
	t.Cleanup(func() {
		n1.s.CloseRingPop()
		n2.s.CloseRingPop()
		time.Sleep(500 * time.Millisecond)
	})

	eventually(t, func() bool {
		m1, err1 := n1.s.ringpop.GetReachableMembers()
		m2, err2 := n2.s.ringpop.GetReachableMembers()
		return err1 == nil && err2 == nil && len(m1) == 2 && len(m2) == 2
	}, "nodes did not form a two-node ring")
	require.Equal(t, addr1, n1.s.address)
	require.Equal(t, addr2, n2.s.address)
	return n1, n2
}

// entityOwnedBy returns an entity ID that both nodes agree belongs to owner.
func entityOwnedBy(t *testing.T, owner *ringpopNode, other *ringpopNode) string {
	t.Helper()
	for i := 0; i < 1000; i++ {
		id := fmt.Sprintf("app.%d", i)
		o1, err1 := owner.s.ringpop.Lookup(id)
		o2, err2 := other.s.ringpop.Lookup(id)
		if err1 == nil && err2 == nil && o1 == owner.s.address && o2 == owner.s.address {
			return id
		}
	}
	t.Fatalf("no entity hashes to %s", owner.s.address)
	return ""
}

// Reproduces the state after a falsely evicted node rejoins: the peer that
// adopted its partition and the rejoined owner both run the same poller.
func TestRebalanceOverRingpopResolvesDuplicateAfterRejoin(t *testing.T) {
	n1, n2 := startTwoNodeCluster(t)
	id := entityOwnedBy(t, n2, n1)
	ownersCopy := n2.running(id)
	adoptedCopy := n1.running(id)

	n1.s.rebalanceEntities()

	assert.False(t, n1.isRunning(id), "adopted copy should be released")
	assert.Equal(t, int32(1), adoptedCopy.stopCount())
	assert.Empty(t, n1.dao.statusUpdates(), "releasing node must not overwrite the owner's record")

	assert.True(t, n2.isRunning(id))
	assert.Equal(t, int32(0), ownersCopy.initCount(), "owner's poller must not be restarted")
	assert.Equal(t, int32(0), ownersCopy.stopCount())
	_, recreated := n2.factory.createdEntity(id)
	assert.False(t, recreated)
	assert.Equal(t, []statusUpdate{{id: id, node: n2.s.address, status: RUNNING}}, n2.dao.statusUpdates())
	assert.Empty(t, n2.factory.retriever.recorded(), "nothing to reconcile when the owner never stopped")
}

func TestRebalanceOverRingpopStartsEntityOnOwnerBeforeReleasingIt(t *testing.T) {
	n1, n2 := startTwoNodeCluster(t)
	id := entityOwnedBy(t, n2, n1)
	misplaced := n1.running(id)

	n1.s.rebalanceEntities()

	assert.False(t, n1.isRunning(id))
	assert.Equal(t, int32(1), misplaced.stopCount())
	assert.True(t, n2.isRunning(id), "owner should now run the entity")
	created, ok := n2.factory.createdEntity(id)
	require.True(t, ok)
	eventually(t, func() bool { return created.initCount() == 1 }, "owner never started the entity")
	assert.Equal(t, []statusUpdate{{id: id, node: n2.s.address, status: RUNNING}}, n2.dao.statusUpdates())
	assert.Len(t, n2.factory.retriever.recorded(), 2, "offset 1 reconciles the current and previous minute")
}

func TestRebalanceOverRingpopLeavesOwnedEntitiesAlone(t *testing.T) {
	n1, n2 := startTwoNodeCluster(t)
	id := entityOwnedBy(t, n1, n2)
	entity := n1.running(id)

	n1.s.rebalanceEntities()

	assert.True(t, n1.isRunning(id))
	assert.Equal(t, int32(0), entity.stopCount())
	assert.False(t, n2.isRunning(id))
	assert.Empty(t, n2.dao.statusUpdates())
}

func TestBootKeepsEntityAlreadyTakenOverWithoutRestart(t *testing.T) {
	n1, n2 := startTwoNodeCluster(t)
	id := entityOwnedBy(t, n2, n1)
	takenOver := n2.running(id)

	err := n2.s.BootEntity(e.EntityInfo{Id: id, Node: n2.s.address, Status: RUNNING}, false)

	require.NoError(t, err)
	assert.True(t, n2.isRunning(id))
	time.Sleep(20 * time.Millisecond)
	assert.Equal(t, int32(0), takenOver.initCount(), "boot must not restart a poller handed over by a peer")
	assert.Equal(t, int32(0), takenOver.stopCount())
	_, recreated := n2.factory.createdEntity(id)
	assert.False(t, recreated)
	assert.Equal(t, []statusUpdate{{id: id, node: n2.s.address, status: RUNNING}}, n2.dao.statusUpdates())
}

// Boot reads the entity table once, so a record can still name the peer that
// released the entity during the handoff. Boot must still end with exactly one
// running copy on the owner.
func TestBootWithStaleRecordEndsWithOneCopyOnOwner(t *testing.T) {
	n1, n2 := startTwoNodeCluster(t)
	id := entityOwnedBy(t, n2, n1)
	n1.running(id)
	n1.s.rebalanceEntities()
	require.False(t, n1.isRunning(id))
	require.True(t, n2.isRunning(id))

	err := n2.s.BootEntity(e.EntityInfo{Id: id, Node: n1.s.address, Status: RUNNING}, false)

	require.NoError(t, err)
	assert.False(t, n1.isRunning(id))
	assert.True(t, n2.isRunning(id))
	assert.Equal(t, 1, n2.s.entities.Count())
	updates := n2.dao.statusUpdates()
	require.NotEmpty(t, updates)
	assert.Equal(t, statusUpdate{id: id, node: n2.s.address, status: RUNNING}, updates[len(updates)-1],
		"the owner's record must end as running")
}

func TestTakeOverRequestOverRingpopIsCheckedAgainstReceiversRing(t *testing.T) {
	n1, n2 := startTwoNodeCluster(t)
	ownedByN1 := entityOwnedBy(t, n1, n2)
	ownedByN2 := entityOwnedBy(t, n2, n1)

	accepted, err := n1.s.requestTakeover(n2.s.address, []string{ownedByN1, ownedByN2})

	require.NoError(t, err)
	assert.Equal(t, []string{ownedByN2}, accepted)
	assert.False(t, n2.isRunning(ownedByN1), "receiver must refuse entities it does not own")
	assert.True(t, n2.isRunning(ownedByN2))
}

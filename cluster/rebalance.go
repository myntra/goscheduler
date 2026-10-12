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
	json2 "encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/golang/glog"
	e "github.com/myntra/goscheduler/cluster_entity"
	"github.com/uber/tchannel-go/json"
)

// Ringpop only reports ServersRemoved to the nodes that observed the removal.
// When a node is falsely declared dead and then rejoins, the other nodes have
// already adopted its entities while the node itself kept running them, so
// both copies poll the same partitions. Rebalancing hands every entity a node
// no longer owns back to its owner and releases it only after the owner
// confirms it is running it, so a handoff can leave a temporary duplicate but
// never a partition that nobody polls.

func (s *Supervisor) lookupOwner(id string) (string, error) {
	return s.ringpop.Lookup(id)
}

// requestRebalance schedules a rebalance without blocking; requests made while
// one is already pending are coalesced into it.
func (s *Supervisor) requestRebalance() {
	select {
	case s.rebalanceTrigger <- struct{}{}:
	default:
	}
}

func (s *Supervisor) startRebalancer() {
	if !s.opt.rebalanceEnabled {
		glog.Infof("Rebalancing is disabled on %s", s.address)
		return
	}
	s.rebalanceStart.Do(func() {
		go s.runRebalancer()
	})
}

// stopRebalancer stops future sweeps without waiting for an in-flight one,
// since a handoff RPC can take several seconds and must not delay shutdown.
func (s *Supervisor) stopRebalancer() {
	s.rebalanceClose.Do(func() {
		close(s.rebalanceStop)
	})
	s.rebalanceStart.Do(func() {
		close(s.rebalanceDone)
	})
}

func (s *Supervisor) runRebalancer() {
	defer close(s.rebalanceDone)

	var tick <-chan time.Time
	if s.opt.rebalanceInterval > 0 {
		ticker := time.NewTicker(s.opt.rebalanceInterval)
		defer ticker.Stop()
		tick = ticker.C
	}

	for {
		select {
		case <-s.rebalanceStop:
			return
		case <-tick:
		case <-s.rebalanceTrigger:
			if !s.waitForRingToSettle() {
				return
			}
		}
		s.rebalanceEntities()
	}
}

// waitForRingToSettle waits out the settle delay and absorbs ring changes that
// arrived meanwhile. It returns false if the rebalancer was stopped.
func (s *Supervisor) waitForRingToSettle() bool {
	if s.opt.rebalanceSettleDelay > 0 {
		timer := time.NewTimer(s.opt.rebalanceSettleDelay)
		select {
		case <-s.rebalanceStop:
			timer.Stop()
			return false
		case <-timer.C:
		}
	}
	select {
	case <-s.rebalanceTrigger:
	default:
	}
	return true
}

// rebalanceEntities hands each locally running entity that the ring assigns to
// another node over to that node, and releases the ones it accepts. Entities
// whose handoff fails keep running here and are retried on the next sweep.
func (s *Supervisor) rebalanceEntities() {
	defer func() {
		if r := recover(); r != nil {
			glog.Errorf("Rebalancing entities on %s failed: %v", s.address, r)
		}
	}()

	misplaced := make(map[string][]string)
	for _, id := range s.entities.Keys() {
		owner, err := s.ownerOf(id)
		if err != nil {
			glog.Errorf("Owner lookup for entity %s on %s failed, keeping it running: %v", id, s.address, err)
			continue
		}
		if owner != s.address {
			misplaced[owner] = append(misplaced[owner], id)
		}
	}

	for owner, ids := range misplaced {
		select {
		case <-s.rebalanceStop:
			return
		default:
		}

		glog.Infof("Handing off %d entities from %s to owner %s", len(ids), s.address, owner)
		accepted, err := s.handoff(owner, ids)
		if err != nil {
			glog.Errorf("Handing off %d entities from %s to %s failed, keeping them running: %v", len(ids), s.address, owner, err)
			continue
		}

		requested := make(map[string]bool, len(ids))
		for _, id := range ids {
			requested[id] = true
		}
		for _, id := range accepted {
			if requested[id] {
				s.releaseEntity(id, owner)
			}
		}
	}
}

// releaseEntity stops a locally running entity that owner has taken over. It
// leaves the cluster DB untouched because owner has already recorded itself as
// the running node. The ring is checked again so an entity whose ownership
// moved during the handoff is kept.
func (s *Supervisor) releaseEntity(id string, owner string) bool {
	defer s.lockEntity(id)()

	current, err := s.ownerOf(id)
	if err != nil || current != owner {
		glog.Infof("Keeping entity %s on %s: owner is now %q (err: %v), expected %s", id, s.address, current, err, owner)
		return false
	}

	if !s.removeEntityLocked(id) {
		return false
	}
	glog.Infof("Released entity %s on %s to its owner %s", id, s.address, owner)
	return true
}

// requestTakeover asks destNode to take ownership of ids and returns the ones it accepted.
func (s *Supervisor) requestTakeover(destNode string, ids []string) ([]string, error) {
	handle, err := s.forwardEntity(nil, Request{
		entity:   EntityIDs{Ids: ids},
		method:   TakeOverEntities,
		destNode: destNode,
	})
	if err != nil {
		return nil, err
	}

	var response Response
	if err := json2.Unmarshal(handle, &response); err != nil {
		return nil, fmt.Errorf("unmarshalling takeover response %s: %w", handle, err)
	}
	if response.Status != SUCCESS {
		return nil, errors.New(response.Error)
	}
	return response.Accepted, nil
}

// TakeOverEntities claims each entity that this node owns according to its own
// view of the ring and reports the claimed IDs back, so the caller only stops
// what this node is now running.
func (s *Supervisor) TakeOverEntities(ctx json.Context, request *EntityIDs) (*Response, error) {
	glog.Infof("TakeOverEntities called on %s for %d entities", s.address, len(request.Ids))
	response := &Response{
		ServerAddress: s.address,
		Status:        SUCCESS,
	}

	for _, id := range request.Ids {
		if err := s.claimEntity(id); err != nil {
			glog.Infof("Rejected takeover of entity %s on %s: %v", id, s.address, err)
			continue
		}
		response.Accepted = append(response.Accepted, id)
	}
	return response, nil
}

// claimEntity makes this node the recorded runner of an entity it owns. An
// entity that is already running is only re-recorded, not restarted. A newly
// started one reconciles recent schedules, as when adopting a removed node's
// entities, to cover buckets the previous runner may have skipped.
func (s *Supervisor) claimEntity(id string) (err error) {
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("claiming entity %s panicked: %v", id, r)
		}
	}()

	owner, err := s.ownerOf(id)
	if err != nil {
		return err
	}
	if owner != s.address {
		return fmt.Errorf("entity %s belongs to %s", id, owner)
	}

	entity := e.EntityInfo{Id: id}
	app, err := s.clusterDao.GetApp(entity.GetAppName())
	if err != nil {
		return err
	}

	if !app.Active {
		glog.Infof("Not starting entity %s on %s as its app is not active", id, s.address)
		return nil
	}

	started, err := s.ensureEntityRunning(id)
	if err != nil {
		return err
	}
	if started {
		glog.Infof("Took over entity %s on %s", id, s.address)
		if s.opt.reconciliationEnabled {
			s.fetchAndRetrySchedule(app, entity.GetPartitionId(), s.opt.reconciliationOffset)
		}
	}
	return nil
}

// ensureEntityRunning starts an entity that is not running on this node and
// records it as running here. An entity that is already running is only
// re-recorded, so its poller is not restarted. It reports whether the entity
// was started.
func (s *Supervisor) ensureEntityRunning(id string) (bool, error) {
	defer s.lockEntity(id)()

	if _, exists := s.entities.Get(id); exists {
		return false, s.clusterDao.UpdateEntityStatus(id, s.address, RUNNING)
	}
	_, err := s.startEntityLocked(id)
	return true, err
}

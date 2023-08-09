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
	e "github.com/myntra/goscheduler/cluster_entity"
	"github.com/myntra/goscheduler/dao"
	"github.com/myntra/goscheduler/poller"
	"github.com/myntra/goscheduler/store"
	"github.com/stretchr/testify/assert"
	"testing"
	"time"
)

func TestNewSupervisorWithRingpop(t *testing.T) {
	supervisor := NewSupervisor(
		new(poller.PollerFactory),
		new(dao.ClusterDaoImplCassandra),
		nil,
		WithClusterName("test"),
		WithAddress("127.0.0.1:2381"),
		WithBootStrapServers([]string{"127.0.0.1:2381"}),
		WithJoinSize(1),
		WithLogEnabled(false),
		WithReplicaPoints(1))

	supervisor.InitRingPop()
	time.Sleep(time.Second)
	nodes, _ := supervisor.ringpop.GetReachableMembers()
	assert.Equal(t, 1, len(nodes))
	supervisor.CloseRingPop()
}

func TestSupervisor_StartEntity(t *testing.T) {
	supervisor := NewSupervisor(
		new(poller.DummyFactory),
		new(dao.DummyClusterDaoImpl),
		nil,
		WithClusterName("test"),
		WithAddress("127.0.0.1:2381"),
		WithBootStrapServers([]string{"127.0.0.1:2381"}),
		WithJoinSize(1),
		WithLogEnabled(false),
		WithReplicaPoints(1))

	for _, test := range []struct {
		Id                string
		RecoverableEntity *RecoverableEntity
	}{
		{
			"abc.1",
			&RecoverableEntity{
				Obj: poller.Dummy{
					AppName:     "abc",
					PartitionId: 1,
				},
				Recovered: 0,
			},
		},
		{
			"abc.2",
			nil,
		},
	} {
		if test.RecoverableEntity != nil {
			supervisor.entities.Set(test.Id, *test.RecoverableEntity)
		}

		done, _ := supervisor.StartEntity(test.Id)
		assert.True(t, done)
	}
}

func TestSupervisor_StopEntity(t *testing.T) {
	supervisor := NewSupervisor(
		new(poller.DummyFactory),
		new(dao.DummyClusterDaoImpl),
		nil,
		WithClusterName("test"),
		WithAddress("127.0.0.1:2381"),
		WithBootStrapServers([]string{"127.0.0.1:2381"}),
		WithJoinSize(1),
		WithLogEnabled(false),
		WithReplicaPoints(1))

	for _, test := range []struct {
		Id                string
		RecoverableEntity *RecoverableEntity
		Exists            bool
	}{
		{
			"abc.1",
			&RecoverableEntity{
				Obj: poller.Dummy{
					AppName:     "abc",
					PartitionId: 1,
				},
				Recovered: 0,
			},
			true,
		},
		{
			"abc.2",
			nil,
			false,
		},
	} {
		if test.RecoverableEntity != nil {
			supervisor.entities.Set(test.Id, *test.RecoverableEntity)
		}

		exists, _ := supervisor.StopEntity(test.Id)
		assert.Equal(t, test.Exists, exists)
		assert.Equal(t, 0, supervisor.entities.Count())
	}
}

func TestSupervisor_ActivateApp(t *testing.T) {
	supervisor := NewSupervisor(
		new(poller.DummyFactory),
		new(dao.DummyClusterDaoImpl),
		nil,
		WithClusterName("test"),
		WithAddress("127.0.0.1:2381"),
		WithBootStrapServers([]string{"127.0.0.1:2381"}),
		WithJoinSize(1),
		WithLogEnabled(false),
		WithReplicaPoints(1))

	supervisor.InitRingPop()
	time.Sleep(time.Second)

	for _, test := range []struct {
		App      store.App
		PollerId string
	}{
		{
			store.App{
				AppId:      "Tony",
				Partitions: 1,
				Active:     true,
			},
			"Tony.0",
		},
		{
			store.App{
				AppId:      "Steve",
				Partitions: 1,
				Active:     true,
			},
			"Steve.0",
		},
	} {

		supervisor.ActivateApp(test.App)

		_, exists := supervisor.entities.Get(test.PollerId)
		assert.True(t, exists)
	}

	supervisor.CloseRingPop()
	time.Sleep(time.Second)
}

func TestSupervisor_DeactivateApp(t *testing.T) {
	supervisor := NewSupervisor(
		new(poller.DummyFactory),
		new(dao.DummyClusterDaoImpl),
		nil,
		WithClusterName("test"),
		WithAddress("127.0.0.1:2381"),
		WithBootStrapServers([]string{"127.0.0.1:2381"}),
		WithJoinSize(1),
		WithLogEnabled(false),
		WithReplicaPoints(1))

	supervisor.InitRingPop()
	time.Sleep(time.Second)

	for _, test := range []struct {
		App               store.App
		RecoverableEntity *RecoverableEntity
		PollerId          string
	}{
		{
			store.App{
				AppId:      "Tony",
				Partitions: 1,
				Active:     true,
			},
			&RecoverableEntity{
				Obj: poller.Dummy{
					AppName:     "Tony",
					PartitionId: 0,
				},
				Recovered: 0,
			},
			"Tony.0",
		},
		{
			store.App{
				AppId:      "Steve",
				Partitions: 1,
				Active:     true,
			},
			&RecoverableEntity{
				Obj: poller.Dummy{
					AppName:     "Steve",
					PartitionId: 0,
				},
				Recovered: 0,
			},
			"Steve.0",
		},
	} {
		if test.RecoverableEntity != nil {
			supervisor.entities.Set(test.PollerId, *test.RecoverableEntity)
		}

		supervisor.DeactivateApp(test.App)

		_, exists := supervisor.entities.Get(test.PollerId)
		assert.False(t, exists)
	}

	supervisor.CloseRingPop()
	time.Sleep(time.Second)
}

func TestSupervisor_StartEntities(t *testing.T) {
	s1 := NewSupervisor(
		new(poller.DummyFactory),
		new(dao.DummyClusterDaoImpl),
		nil,
		WithClusterName("test"),
		WithAddress("127.0.0.1:2381"),
		WithBootStrapServers([]string{"127.0.0.1:2381"}),
		WithJoinSize(1),
		WithLogEnabled(false),
		WithReplicaPoints(1))

	s2 := NewSupervisor(
		new(poller.DummyFactory),
		new(dao.DummyClusterDaoImpl),
		nil,
		WithClusterName("test"),
		WithAddress("127.0.0.1:2382"),
		WithBootStrapServers([]string{"127.0.0.1:2382"}),
		WithJoinSize(1),
		WithLogEnabled(false),
		WithReplicaPoints(1))

	s1.InitRingPop()
	time.Sleep(time.Second)

	s2.InitRingPop()
	time.Sleep(time.Second)

	for _, test := range []struct {
		EntityIds *EntityIDs
		PollerId  string
	}{
		{
			&EntityIDs{Ids: []string{"Tony.0"}},
			"Tony.0",
		},
		{
			&EntityIDs{Ids: []string{"Steve.0"}},
			"Steve.0",
		},
	} {

		_, exists := s1.entities.Get(test.PollerId)
		assert.False(t, exists)

		res, _ := s1.StartEntities(nil, test.EntityIds)
		assert.Equal(t, SUCCESS, res.Status)
		assert.Equal(t, "", res.Error)

		_, exists = s1.entities.Get(test.PollerId)
		assert.True(t, exists)
	}

	s1.CloseRingPop()
	time.Sleep(time.Second)

	s2.CloseRingPop()
	time.Sleep(time.Second)
}

func TestSupervisor_StopEntities(t *testing.T) {
	s1 := NewSupervisor(
		new(poller.DummyFactory),
		new(dao.DummyClusterDaoImpl),
		nil,
		WithClusterName("test"),
		WithAddress("127.0.0.1:2381"),
		WithBootStrapServers([]string{"127.0.0.1:2381"}),
		WithJoinSize(1),
		WithLogEnabled(false),
		WithReplicaPoints(1))

	s2 := NewSupervisor(
		new(poller.DummyFactory),
		new(dao.DummyClusterDaoImpl),
		nil,
		WithClusterName("test"),
		WithAddress("127.0.0.1:2382"),
		WithBootStrapServers([]string{"127.0.0.1:2382"}),
		WithJoinSize(1),
		WithLogEnabled(false),
		WithReplicaPoints(1))

	s1.InitRingPop()
	time.Sleep(time.Second)

	s2.InitRingPop()
	time.Sleep(time.Second)

	for _, test := range []struct {
		RecoverableEntity *RecoverableEntity
		EntityIds         *EntityIDs
		PollerId          string
	}{
		{
			&RecoverableEntity{
				Obj: poller.Dummy{
					AppName:     "Tony",
					PartitionId: 0,
				},
				Recovered: 0,
			},
			&EntityIDs{Ids: []string{"Tony.0"}},
			"Tony.0",
		},
		{
			nil,
			&EntityIDs{Ids: []string{"Steve.0"}},
			"Steve.0",
		},
	} {

		if test.RecoverableEntity != nil {
			s1.entities.Set(test.PollerId, *test.RecoverableEntity)
		}

		res, _ := s1.StopEntities(nil, test.EntityIds)
		assert.Equal(t, SUCCESS, res.Status)
		assert.Equal(t, "", res.Error)

		_, exists := s1.entities.Get(test.PollerId)
		assert.False(t, exists)
	}

	s1.CloseRingPop()
	time.Sleep(time.Second)

	s2.CloseRingPop()
	time.Sleep(time.Second)
}

func TestSupervisor_BootEntity(t *testing.T) {
	s := NewSupervisor(
		new(poller.DummyFactory),
		new(dao.DummyClusterDaoImpl),
		nil,
		WithClusterName("test"),
		WithAddress("127.0.0.1:2381"),
		WithBootStrapServers([]string{"127.0.0.1:2381"}),
		WithJoinSize(1),
		WithLogEnabled(false),
		WithReplicaPoints(1))

	s.InitRingPop()

	for _, test := range []struct {
		EntityInfo e.EntityInfo
		PollerId   string
	}{
		{
			e.EntityInfo{
				Id:      "Tony.0",
				Node:    s.address,
				Status:  0,
				History: "",
			},
			"Tony.0",
		},
		{
			e.EntityInfo{
				Id:      "Steve.0",
				Node:    s.address,
				Status:  1,
				History: "",
			},
			"Steve.0",
		},
	} {

		if err := s.BootEntity(test.EntityInfo, false); err != nil {
			t.Errorf("Got err: %v", err)
		}

		_, exists := s.entities.Get(test.PollerId)
		assert.True(t, exists)
	}
	s.CloseRingPop()
	time.Sleep(time.Second)
}

func TestSupervisor_ForwardOrPanic(t *testing.T) {
	s1 := NewSupervisor(
		new(poller.DummyFactory),
		new(dao.DummyClusterDaoImpl),
		nil,
		WithClusterName("test"),
		WithAddress("127.0.0.1:2381"),
		WithBootStrapServers([]string{"127.0.0.1:2381"}),
		WithJoinSize(1),
		WithLogEnabled(false),
		WithReplicaPoints(1))

	s2 := NewSupervisor(
		new(poller.DummyFactory),
		new(dao.DummyClusterDaoImpl),
		nil,
		WithClusterName("test"),
		WithAddress("127.0.0.1:2382"),
		WithBootStrapServers([]string{"127.0.0.1:2382"}),
		WithJoinSize(1),
		WithLogEnabled(false),
		WithReplicaPoints(1))

	s1.InitRingPop()
	time.Sleep(time.Second)

	s2.InitRingPop()
	time.Sleep(time.Second)

	// 1. Forward entities from s1 and start on s2 node
	// 2. Forward entities from s1 and stop on s2 node
	for _, test := range []struct {
		EntityInfo e.EntityInfo
		PollerId   string
	}{
		{
			e.EntityInfo{
				Id:      "Tony.0",
				Node:    s2.address,
				Status:  1,
				History: "",
			},
			"Tony.0",
		},
		{
			e.EntityInfo{
				Id:      "Steve.0",
				Node:    s2.address,
				Status:  1,
				History: "",
			},
			"Steve.0",
		},
	} {
		s1.forwardOrPanic(test.EntityInfo, StartEntities)
		_, exists := s2.entities.Get(test.PollerId)
		assert.True(t, exists)

		s1.forwardOrPanic(test.EntityInfo, StopEntities)
		_, exists = s2.entities.Get(test.PollerId)
		assert.False(t, exists)
	}

	s1.CloseRingPop()
	time.Sleep(time.Second)

	s2.CloseRingPop()
	time.Sleep(time.Second)
}

func TestSupervisor_appDetailsUpdateBroadcast(t *testing.T) {
	s1 := NewSupervisor(
		new(poller.DummyFactory),
		new(dao.DummyClusterDaoImpl),
		nil,
		WithClusterName("test"),
		WithAddress("127.0.0.1:2381"),
		WithBootStrapServers([]string{"127.0.0.1:2381"}),
		WithJoinSize(1),
		WithLogEnabled(false),
		WithReplicaPoints(1))

	s2 := NewSupervisor(
		new(poller.DummyFactory),
		new(dao.DummyClusterDaoImpl),
		nil,
		WithClusterName("test"),
		WithAddress("127.0.0.1:2382"),
		WithBootStrapServers([]string{"127.0.0.1:2382"}),
		WithJoinSize(1),
		WithLogEnabled(false),
		WithReplicaPoints(1))

	s1.InitRingPop()
	time.Sleep(time.Second)

	s2.InitRingPop()
	time.Sleep(time.Second)

	for _, test := range []struct {
		Name string
	}{
		{"Tony"},
		{"Steve"},
		{"Thor"},
	} {

		// Should not panic
		s1.appDetailsUpdateBroadcast(test.Name)
	}

	s1.CloseRingPop()
	time.Sleep(time.Second)

	s2.CloseRingPop()
	time.Sleep(time.Second)
}

func TestSupervisor_Boot(t *testing.T) {
	s := NewSupervisor(
		new(poller.DummyFactory),
		new(dao.DummyClusterDaoImpl),
		nil,
		WithClusterName("test"),
		WithAddress("127.0.0.1:2381"),
		WithBootStrapServers([]string{"127.0.0.1:2381"}),
		WithJoinSize(1),
		WithLogEnabled(false),
		WithReplicaPoints(1))

	s.InitRingPop()
	time.Sleep(time.Second)

	// 1. Forward entities from s1 and start on s2 node
	// 2. Forward entities from s1 and stop on s2 node
	for _, test := range []struct {
		PollerId string
	}{
		{"Tony.0"},
		{"Steve.0"},
		{"Thor.0"},
	} {
		// Should not panic
		s.Boot()
		_, exists := s.entities.Get(test.PollerId)
		assert.True(t, exists)
	}

	s.CloseRingPop()
	time.Sleep(time.Second)
}

func TestSupervisor_StopNode(t *testing.T) {
	s := NewSupervisor(
		new(poller.DummyFactory),
		new(dao.DummyClusterDaoImpl),
		nil,
		WithClusterName("test"),
		WithAddress("127.0.0.1:2381"),
		WithBootStrapServers([]string{"127.0.0.1:2381"}),
		WithJoinSize(1),
		WithLogEnabled(false),
		WithReplicaPoints(1))

	s.InitRingPop()
	time.Sleep(time.Second)

	entity := map[string]interface{}{
		"Tony.0": RecoverableEntity{
			Obj: poller.Dummy{
				AppName:     "Tony",
				PartitionId: 0,
			},
			Recovered: 0,
		},
		"Steve.0": RecoverableEntity{
			Obj: poller.Dummy{
				AppName:     "Steve",
				PartitionId: 0,
			},
			Recovered: 0,
		},
		"Thor.0": RecoverableEntity{
			Obj: poller.Dummy{
				AppName:     "Thor",
				PartitionId: 0,
			},
			Recovered: 0,
		},
	}

	s.entities.MSet(entity)

	// Stop all the entities
	s.StopNode()
	assert.Equal(t, 0, s.entities.Count())

	s.CloseRingPop()
	time.Sleep(time.Second)
}

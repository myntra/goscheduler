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
	"github.com/golang/glog"
	e "github.com/myntra/goscheduler/cluster_entity"
	"github.com/myntra/goscheduler/constants"
	"github.com/myntra/goscheduler/dao"
	p "github.com/myntra/goscheduler/monitoring"
	"github.com/myntra/goscheduler/store"
	cmap "github.com/orcaman/concurrent-map"
	"github.com/sirupsen/logrus"
	"github.com/uber-common/bark"
	"github.com/uber/ringpop-go"
	"github.com/uber/ringpop-go/discovery/statichosts"
	"github.com/uber/ringpop-go/events"
	"github.com/uber/ringpop-go/forward"
	"github.com/uber/ringpop-go/hashring"
	"github.com/uber/ringpop-go/logging"
	"github.com/uber/ringpop-go/membership"
	"github.com/uber/ringpop-go/swim"
	"github.com/uber/tchannel-go"
	"github.com/uber/tchannel-go/json"
	"golang.org/x/net/context"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"
)

const (
	StartEntities    = "StartEntities"
	StopEntities     = "StopEntities"
	AppDetailsUpdate = "AppDetailsUpdate"
)

const (
	STOPPED = iota
	RUNNING
)

// noinspection SpellCheckingInspection
type Supervisor struct {
	opt           options
	clusterName   string
	address       string
	ringpop       *ringpop.Ringpop
	channel       *tchannel.Channel
	entities      cmap.ConcurrentMap
	entityFactory e.EntityFactory
	clusterDao    dao.ClusterDao
	scheduleDao   dao.ScheduleDao
	monitor       p.Monitor
}

type options struct {
	clusterName           string
	address               string
	logEnabled            bool
	logLevels             map[string]logging.Level
	replicaPoints         int
	bootStrapServers      []string
	joinSize              int
	statsD                bark.StatsReporter
	reconciliationEnabled bool
	reconciliationOffset  int
}

var defaultOptions = options{
	clusterName: "default",
	address:     "127.0.0.1:9091",
	logEnabled:  true,
	logLevels: map[string]logging.Level{
		"join":          logging.Debug,
		"damping":       logging.Debug,
		"dissemination": logging.Debug,
		"gossip":        logging.Debug,
		"membership":    logging.Debug,
		"ring":          logging.Debug,
		"suspicion":     logging.Debug,
	},
	replicaPoints:         2,
	bootStrapServers:      []string{"127.0.0.1:9091"},
	joinSize:              1,
	reconciliationEnabled: true,
	reconciliationOffset:  3,
}

type Option func(*options)

func WithClusterName(clusterName string) Option {
	return func(o *options) {
		o.clusterName = clusterName
	}
}

func WithAddress(address string) Option {
	return func(o *options) {
		o.address = address
	}
}

func WithLogEnabled(logEnabled bool) Option {
	return func(o *options) {
		o.logEnabled = logEnabled
	}
}

func WithLogLevels(logLevels map[string]logging.Level) Option {
	return func(o *options) {
		o.logLevels = logLevels
	}
}

func WithReplicaPoints(replicaPoints int) Option {
	return func(o *options) {
		o.replicaPoints = replicaPoints
	}
}

func WithBootStrapServers(bootStrapServers []string) Option {
	return func(o *options) {
		o.bootStrapServers = bootStrapServers
	}
}

func WithJoinSize(joinSize int) Option {
	return func(o *options) {
		o.joinSize = joinSize
	}
}

func WithStatsD(statsD bark.StatsReporter) Option {
	return func(o *options) {
		o.statsD = statsD
	}
}

func WithReconciliationEnabled(reconciliationEnabled bool) Option {
	return func(o *options) {
		o.reconciliationEnabled = reconciliationEnabled
	}
}

func WithReconciliationOffset(reconciliationOffset int) Option {
	return func(o *options) {
		o.reconciliationOffset = reconciliationOffset
	}
}

func NewSupervisor(entityFactory e.EntityFactory, clusterDao dao.ClusterDao, monitor p.Monitor, opt ...Option) *Supervisor {
	opts := defaultOptions
	for _, o := range opt {
		o(&opts)
	}

	glog.Infof("Starting supervisor with options : %+v", opts)

	supervisor := &Supervisor{
		opt:           opts,
		clusterName:   opts.clusterName,
		entities:      cmap.New(),
		entityFactory: entityFactory,
		clusterDao:    clusterDao,
		monitor:       monitor,
	}

	return supervisor
}

// initRingPop initializes ringpop with provided configurations
func (s *Supervisor) InitRingPop() {
	var err error

	glog.Infof("Supervisor: %+v", s)
	s.channel, err = tchannel.NewChannel(s.opt.clusterName, nil)
	if err != nil {
		panic("channel did not create successfully")
	}
	glog.Infof("TChannel created %+v", s.channel)

	options := []ringpop.Option{
		ringpop.Channel(s.channel),
		ringpop.Address(s.opt.address),
		ringpop.HashRingConfig(&hashring.Configuration{
			ReplicaPoints: s.opt.replicaPoints,
		})}

	if s.opt.logEnabled {
		logger := logrus.New()
		logger.Out = os.Stdout
		options = append(options,
			ringpop.Logger(bark.NewLoggerFromLogrus(logger)),
			ringpop.LogLevels(s.opt.logLevels),
			ringpop.Statter(s.opt.statsD))
	}

	glog.Infof("ringpop options: %+v", options)
	s.ringpop, err = ringpop.New(s.opt.clusterName, options...)
	if err != nil {
		panic(fmt.Sprintf("Ringpop cluster creation failed with error %v", err))
	}

	glog.Infof("Ringpop cluster created %+v", s.ringpop)

	if err = s.channel.ListenAndServe(s.opt.address); err != nil {
		panic(fmt.Sprintf("Could not listen on given host port: %v", err))
	}

	s.ringpop.AddListener(s)

	bootstrapOpts := &swim.BootstrapOptions{
		DiscoverProvider: statichosts.New(s.opt.bootStrapServers...),
		JoinSize:         s.opt.joinSize,
	}

	glog.Infof("Supervisor: %+v", s.ringpop)
	if _, err = s.ringpop.Bootstrap(bootstrapOpts); err != nil {
		panic(fmt.Sprintf("Ringpop bootstrap failed: %v", err))
	}

	if err := s.RegisterHandler(); err != nil {
		panic(fmt.Sprintf("Error while registering handler %+v", err))
	}

	s.address, err = s.ringpop.WhoAmI()
	if err != nil {
		panic(fmt.Sprintf("Error initializing ringpop %v", err))
	}
}

// Stop ringpop, TChannel gracefully
func (s *Supervisor) CloseRingPop() {
	s.ringpop.Destroy()
	s.channel.Close()
}

// StartEntity starts an entity if it is not already running.
// Adds entity in in-memory concurrent map if the entity does not exist.
// Updates DB with entity status as Running.
func (s *Supervisor) StartEntity(id string) (bool, error) {
	value, exists := s.entities.Get(id)
	if exists == false {
		entity := s.entityFactory.CreateEntity(id)
		recoverableEntity := RecoverableEntity{Obj: entity}
		s.entities.Set(id, recoverableEntity)
		go recoverableEntity.Start()
	} else {
		recoverableEntity := value.(RecoverableEntity)
		go recoverableEntity.Start()
	}
	return true, s.clusterDao.UpdateEntityStatus(id, s.address, RUNNING)
}

// StopEntity stops an entity if it is running.
// Removes entity from in-memory concurrent map.
// Updates DB with entity status as stopped.
func (s *Supervisor) StopEntity(id string) (bool, error) {
	var err error

	value, exists := s.entities.Get(id)
	if exists {
		recoverableEntity := value.(RecoverableEntity)
		recoverableEntity.Stop()
		s.entities.Remove(id)
		err = s.clusterDao.UpdateEntityStatus(id, s.address, STOPPED)
	}
	return exists, err
}

// StartEntities iterates over the list of entityIds.
// Starts the entities on current node if destination node is same as own address
// or forward the entities to respective destination node
func (s *Supervisor) StartEntities(ctx json.Context, request *EntityIDs) (res *Response, err error) {
	glog.Infof("StartEntities called with arg %+v", request)
	res = &Response{
		ServerAddress: s.address,
		Error:         "",
		Status:        SUCCESS,
	}

	defer func() {
		if r := recover(); r != nil {
			res.Error = r.(error).Error()
			res.Status = FAILED
		}
	}()

	for _, id := range request.Ids {
		destNode, err := s.ringpop.Lookup(id)
		if err != nil {
			panic(err)
		}

		if destNode == s.address {
			if _, err := s.StartEntity(id); err != nil {
				return res, err
			}
			continue
		}

		request := Request{
			entity:   EntityIDs{Ids: []string{id}},
			method:   StartEntities,
			destNode: destNode,
		}
		s.forwardOrPanicIfRequired(ctx, request)
	}
	return res, nil
}

// StopEntities iterates over the list of entityIds.
// Sops the entities on current node if destination node is same as own address
// or forward the entities to respective destination node
func (s *Supervisor) StopEntities(ctx json.Context, request *EntityIDs) (res *Response, err error) {
	glog.Infof("StopEntities called with arg %+v", request)
	res = &Response{
		ServerAddress: s.address,
		Error:         "",
		Status:        SUCCESS,
	}

	defer func() {
		if r := recover(); r != nil {
			res.Error = r.(error).Error()
			res.Status = FAILED
		}
	}()

	for _, id := range request.Ids {
		exists, err := s.StopEntity(id)
		if err != nil {
			panic(err)
		}

		destNode, err := s.ringpop.Lookup(id)
		if err != nil {
			panic(err)
		}

		if exists == false {

			// TODO: Do we need this?
			// Since exists is false entity does not belong to this node
			// it can be simply forwarded
			if s.address == destNode {
				continue
			}

			request := Request{
				entity:   EntityIDs{Ids: []string{id}},
				method:   StopEntities,
				destNode: destNode,
			}
			s.forwardOrPanicIfRequired(ctx, request)
		}

	}
	return res, nil
}

// forwardEntity forwards the entityId request to destNode
func (s *Supervisor) forwardEntity(ctx json.Context, r Request) ([]byte, error) {
	var newRequest interface{}
	var ids []string
	headers, _ := json2.Marshal(make(map[string]string))

	if ctx != nil {
		var err error
		headers, err = json2.Marshal(ctx.Headers())
		if err != nil {
			return nil, err
		}
	}

	forwardOptions := &forward.Options{
		MaxRetries:     3,
		RerouteRetries: false,
		RetrySchedule:  []time.Duration{3 * time.Second, 6 * time.Second, 12 * time.Second},
		Timeout:        3 * time.Second,
		Headers:        headers,
	}

	switch v := r.entity.(type) {
	case EntityIDs:
		ids = v.Ids
		newRequest = &EntityIDs{Ids: ids}
	case AppNames:
		ids = v.Names
		newRequest = &AppNames{Names: ids}
	default:
		return nil, errors.New(fmt.Sprintf("Unknown entity %+v", r.entity))
	}

	bytes, err := json2.Marshal(newRequest)
	if err != nil {
		return nil, errors.New(fmt.Sprintf("Exception while marshalling %+v", newRequest))
	}

	handle, err := s.ringpop.Forward(r.destNode, ids, bytes, s.clusterName, r.method, tchannel.JSON, forwardOptions)
	glog.Info(r.destNode, ids, bytes, s.clusterName, r.method, tchannel.JSON, forwardOptions)
	return handle, err
}

// forward clusterEntity with registered method to entity node
func (s *Supervisor) forward(ctx json.Context, entity e.EntityInfo, method string) ([]byte, error) {
	request := Request{
		entity:   EntityIDs{Ids: []string{entity.Id}},
		method:   method,
		destNode: entity.Node,
	}
	return s.forwardEntity(ctx, request)
}

// forward or panic
func (s *Supervisor) forwardOrPanic(entity e.EntityInfo, method string) {
	request := Request{
		entity:   EntityIDs{Ids: []string{entity.Id}},
		method:   method,
		destNode: entity.Node,
	}
	s.forwardOrPanicIfRequired(nil, request)
}

// forward or panic
// TODO: Change function name
func (s *Supervisor) forwardOrPanicIfRequired(ctx json.Context, r Request) {
	var response Response
	glog.Infof("Forwarding entity %+v to the node %s from node %s", r.entity, r.destNode, s.address)
	handle, err := s.forwardEntity(ctx, r)
	if handle != nil {
		if err = json2.Unmarshal(handle, &response); err != nil {
			panic(errors.New(fmt.Sprintf("Error while Unmarshalling %s response, err: %+v", handle, err)))
		}
	} else if err != nil {
		panic(errors.New(fmt.Sprintf("Forwarding entity %+v to node %s from node %s failed with error : %+v", r.entity, r.destNode, s.address, err.Error())))
	}
	glog.Infof("Forwarding entity %+v to node %s from node %s succeeded with response %+v", r.entity, r.destNode, s.address, response)
}

// appDetailsUpdateBroadcast broadcasts app update message to all other reachable nodes
func (s *Supervisor) appDetailsUpdateBroadcast(appName string) {
	glog.Infof("Broadcasting app update event for app %s", appName)

	reachableNodes, err := s.ringpop.GetReachableMembers()
	if err != nil {
		glog.Errorf("Error getting reachable members %+v", err)
		return
	}

	for _, node := range reachableNodes {
		glog.Infof("Broadcasting app update event for app %s to %s", appName, node)
		if node == s.address {
			s.AppDetailsUpdateHandler(appName)
			continue
		}

		request := Request{
			entity:   AppNames{Names: []string{appName}},
			method:   AppDetailsUpdate,
			destNode: node,
		}
		s.forwardOrPanicIfRequired(nil, request)
	}
}

// AppDetailsUpdateEventHandler receives app update event
// Invalidates cache based on appName
func (s *Supervisor) AppDetailsUpdateEventHandler(ctx json.Context, request *AppNames) (*Response, error) {
	glog.Infof("Called handler for appDetails update broadcast")
	response := Response{
		ServerAddress: s.address,
		Error:         "",
		Status:        SUCCESS,
	}

	defer func() {
		if r := recover(); r != nil {
			response.Error = r.(error).Error()
			response.Status = FAILED
		}
	}()

	for _, appName := range request.Names {
		s.AppDetailsUpdateHandler(appName)
	}

	return &response, nil
}

// Boot fetch all the entities from DB and starts them one by one
// panics and stops the process in case there is any issue in starting any entity
func (s *Supervisor) Boot() {
	for _, entity := range s.clusterDao.GetAllEntitiesInfo() {
		if err := s.BootEntity(entity, false); err != nil {
			panic(err)
		}
	}
}

// StopNode stops all the entities assigned to that node before it is brought down
func (s *Supervisor) StopNode() {
	glog.Info("!!!!!!Inside StopNode!!!!!!")
	for _, id := range s.entities.Keys() {
		if _, err := s.StopEntity(id); err != nil {
			panic(errors.New(fmt.Sprintf("Error while stopping key: %s, error: %+v", id, err)))
		}
	}
}

// BootEntity perform following functions
// 1. Get all reachable members from a node
// 2. Check the destination node for a entity
// 3. Start/forward the entity depending on the destination node
func (s *Supervisor) BootEntity(entity e.EntityInfo, forward bool) (err error) {
	defer func() {
		if r := recover(); r != nil {
			err = r.(error)
			glog.Errorf("Boot entity %+v failed with error %+v", entity, err)
		}
	}()

	appName := entity.GetAppName()
	app, err := s.clusterDao.GetApp(appName)
	if err != nil {
		panic(errors.New(fmt.Sprintf("Error: %+v, while getting app status while booting", err)))
	}
	if app.Active == false {
		glog.Infof("Not activating poller %s for deactivated app %s", entity.Id, app.AppId)
		return nil
	}

	members, err := s.ringpop.GetReachableMembers()
	if err != nil {
		panic(errors.New(fmt.Sprintf("Error %s on node %s while querying for reachable mebers", err.Error(), s.address)))
	}

	glog.Infof("reachableMembers %+v", members)

	// Create a map of reachable nodes
	// A self node is marked as false as we are not required to forward requests to them
	reachableMembers := make(map[string]bool)
	for _, member := range members {
		reachableMembers[member] = true
	}
	reachableMembers[s.address] = false

	// Check which node the current entity belongs to
	destNode, err := s.ringpop.Lookup(entity.Id)
	if err != nil {
		panic(errors.New(fmt.Sprintf("Lookup failed for entity %s with error %+v", entity.Id, err)))
	}

	glog.Infof("destination node %s for entity %s", destNode, entity.Id)

	// If destination node is own address start the entity on current node
	// else forward the entity to destination node if required
	if destNode == s.address {
		if entity.Status == RUNNING && reachableMembers[entity.Node] == true {
			//This case happens when the current node is newly added
			//Then this node will check if the poller is already running on a reachable node
			//If it is, send the message to stop the entity
			//Note: Message to stop the entity is never send to itself as reachableMembers[s.address] is marked as false

			//Is possible if this node was killed abruptly in last run
			s.forwardOrPanic(entity, StopEntities)
		}

		if _, err := s.StartEntity(entity.Id); err != nil {
			panic(errors.New(fmt.Sprintf("Start entity failed for entity %s with error %+v", entity.Id, err)))
		}

		return
	}

	if forward == true {
		s.forwardOrPanic(entity, StartEntities)
	}

	return nil
}

// Offload perform following functions
// 1. Get all entities assigned to the offloaded node
// 2. Check the destination node for a entity
// 3. Start the entity if the current node is the destination node
// 4. Reconcile schedules if there was any miss during offloading
func (s *Supervisor) OffloadOrPanic(nodeName string) {
	glog.Infof("%s offloading %s", s.address, nodeName)

	//Note: Assuming MembersRemoved will be called only if the node is down, which means all entities are stopped on that node
	for _, entity := range s.clusterDao.GetAllEntitiesInfoOfNode(nodeName) {
		appName := entity.GetAppName()
		app, err := s.clusterDao.GetApp(appName)
		if err != nil {
			panic(errors.New(fmt.Sprintf("Could not get app: %s", appName)))
		}
		if !app.Active {
			glog.Infof("Not starting the entity %+v as its app is not active", entity)
			continue
		}

		destNode, err := s.ringpop.Lookup(entity.Id)
		if err != nil {
			panic(errors.New(fmt.Sprintf("Lookup failed with error %s", err)))
		}

		if destNode == s.address {
			glog.Infof("Starting entity %s on %s from old node %s", entity.Id, s.address, nodeName)
			if _, err := s.StartEntity(entity.Id); err != nil {
				panic(errors.New(fmt.Sprintf("Error starting entity: %s", entity.Id)))
			}

			if s.opt.reconciliationEnabled {
				s.fetchAndRetrySchedule(app, entity.GetPartitionId(), s.opt.reconciliationOffset)
			}
		}
	}
}

// AppDetailsUpdateHandler invalidates in memory cache
func (s *Supervisor) AppDetailsUpdateHandler(appName string) {
	s.clusterDao.InvalidateSingleAppCache(appName)
}

// RegisterHandler registers actions against respective methods
func (s *Supervisor) RegisterHandler() error {
	hmap := map[string]interface{}{StartEntities: s.StartEntities, StopEntities: s.StopEntities, AppDetailsUpdate: s.AppDetailsUpdateEventHandler}

	return json.Register(s.channel, hmap, func(ctx context.Context, err error) {
		glog.Errorf("error occurred: %v %+v", err, ctx)
	})
}

// HandleEvent handle different events emitted by Ringpop
// TODO: Check all events
func (s *Supervisor) HandleEvent(event events.Event) {
	switch v := event.(type) {
	case events.RingChangedEvent:
		change := event.(events.RingChangedEvent)
		glog.Infof("Ring updated %+v", v)

		for _, server := range change.ServersRemoved {
			if s.address == server {
				panic("ServersRemoved for Self!!!")
			}
			s.OffloadOrPanic(server)
		}
		break
	case swim.MemberlistChangesReceivedEvent:
		members, _ := s.ringpop.GetReachableMembers()
		glog.Infof("Ring updated %+v", members)
		break
	case membership.ChangeEvent:
	case swim.MakeNodeStatusEvent:
	case swim.PingRequestsSendEvent:
	case swim.ChecksumComputeEvent:
	case swim.MaxPAdjustedEvent:
	case events.RingChecksumEvent:
	case swim.DiscoHealEvent:
	case events.Ready:
	case swim.ProtocolDelayComputeEvent:
	case swim.ProtocolFrequencyEvent:
	case swim.PingSendCompleteEvent:
	case swim.ChangesCalculatedEvent:
	case swim.PingSendEvent:
	case swim.PingReceiveEvent:
	case events.LookupEvent:
	case swim.AttemptHealEvent:
	case swim.JoinCompleteEvent:
	case forward.InflightRequestsChangedEvent:
	case forward.RequestForwardedEvent:
	case forward.SuccessEvent:
	default:
		glog.Errorf("Received unhandled type %T", v)
	}
}

// DeactivateApp stops all the pollers for the app
// Forwards the pollers to stop in case destination node is different
func (s *Supervisor) DeactivateApp(app store.App) {
	glog.Infof("Disabling app %s", app.AppId)
	var partition uint32 = 0
	for ; partition < app.Partitions; partition++ {
		entity := e.EntityInfo{Id: app.AppId + constants.PollerKeySep + strconv.Itoa(int(partition))}
		glog.Infof("Disabling entity %s", entity.Id)
		destNode, err := s.ringpop.Lookup(entity.Id)
		if err != nil {
			panic(errors.New(fmt.Sprintf("Lookup failed with error %s", err)))
		}
		if destNode != s.address {
			entity.Node = destNode
			s.forwardOrPanic(entity, StopEntities)
			continue
		}

		if _, err := s.StopEntity(entity.Id); err != nil {
			panic(errors.New(fmt.Sprintf("Stopping entity failed with error %+v", err)))
		}
	}
	s.appDetailsUpdateBroadcast(app.AppId)
}

// ActivateApp starts all the pollers for the app
// Forwards the pollers to start in case destination node is different
// TODO: Try to combine above methods
func (s Supervisor) ActivateApp(app store.App) {
	glog.Infof("Enabling app %s", app.AppId)
	var partition uint32 = 0
	for ; partition < app.Partitions; partition++ {
		entity := e.EntityInfo{Id: app.AppId + constants.PollerKeySep + strconv.Itoa(int(partition))}
		glog.Infof("Enabling entity %s", entity.Id)
		destNode, err := s.ringpop.Lookup(entity.Id)
		if err != nil {
			panic(errors.New(fmt.Sprintf("Lookup failed with error %s", err)))
		}
		if destNode != s.address {
			entity.Node = destNode
			s.forwardOrPanic(entity, StartEntities)
			continue
		}

		if _, err := s.StartEntity(entity.Id); err != nil {
			panic(errors.New(fmt.Sprintf("Starting entity failed with error %+v", err)))
		}

	}
	s.appDetailsUpdateBroadcast(app.AppId)
}

// Retry a missed schedule based on app, partitionId and timeOffset
func (s *Supervisor) fetchAndRetrySchedule(app store.App, partitionId int, timeOffset int) {
	glog.Infof("Retrying for App:-> %+v", app)
	glog.Infof("App reconcile offset:-> %d", timeOffset)

	year, month, day := time.Now().Date()
	hr, min, _ := time.Now().Clock()
	timeBucket := time.Date(year, month, day, hr, min, 0, 0, time.Now().Location())
	for i := timeOffset; i >= 0; i-- {
		timestamp := timeBucket.Add(time.Duration(-i) * time.Minute)

		scheduleRetrieverImpl := s.entityFactory.GetEntityRetriever(app.AppId)
		if err := scheduleRetrieverImpl.BulkAction(app, partitionId, timestamp, []store.Status{store.Scheduled, store.Miss}, store.Reconcile); err != nil {
			glog.Infof("Error while reconciling for appId: %s, partitionId: %d, timestamp: %+v, err: %s",
				app.AppId,
				partitionId,
				timestamp,
				err.Error())
		}
	}
}

// WaitForTermination waits for OS signals to terminate
// Stops the node, closes stastDClient before exiting the program
func (s *Supervisor) WaitForTermination() {
	c := make(chan os.Signal, 1)
	signal.Notify(c, syscall.SIGTERM, syscall.SIGQUIT, syscall.SIGINT, syscall.SIGABRT, syscall.SIGSEGV)
	exit := make(chan bool, 1)

	go func() {
		glog.Info("Shutting down with Signal:", <-c)
		glog.Flush()
		s.StopNode()
		s.CloseRingPop()
		exit <- true
	}()

	glog.Info("This is before end")
	<-exit
	glog.Info("This is end")
}

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

package service

import (
	"github.com/myntra/goscheduler/cluster"
	c "github.com/myntra/goscheduler/conf"
	"github.com/myntra/goscheduler/constants"
	"github.com/myntra/goscheduler/dao"
	"github.com/myntra/goscheduler/monitoring"
)

type Service struct {
	Config      *c.Configuration
	Supervisor  cluster.SupervisorHandler
	ClusterDao  dao.ClusterDao
	ScheduleDao dao.ScheduleDao
	Monitor     monitoring.Monitor
}

func NewService(config *c.Configuration, supervisor cluster.SupervisorHandler, clusterDao dao.ClusterDao, scheduleDAO dao.ScheduleDao, monitor monitoring.Monitor) *Service {
	return &Service{
		Config:      config,
		Supervisor:  supervisor,
		ClusterDao:  clusterDao,
		ScheduleDao: scheduleDAO,
		Monitor:     monitor,
	}
}

func (s *Service) recordRequestStatus(name, status string) {
	if s.Monitor != nil {
		s.Monitor.IncCounter(constants.RequestStatus, map[string]string{"request": name, "status": status}, 1)
	}
}

func (s *Service) recordRequestAppStatus(name, app, status string) {
	if s.Monitor != nil {
		s.Monitor.IncCounter(constants.RequestAppStatus, map[string]string{"request": name, "appId": app, "status": status}, 1)
	}
}

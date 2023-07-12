package service

import (
	"github.com/myntra/goscheduler/cluster"
	c "github.com/myntra/goscheduler/conf"
	"github.com/myntra/goscheduler/dao"
	"github.com/myntra/goscheduler/monitoring"
)

type Service struct {
	Config      *c.Configuration
	supervisor  cluster.SupervisorHandler
	clusterDao  dao.ClusterDao
	scheduleDao dao.ScheduleDao
	Monitoring  *monitoring.Monitoring
}

func NewService(config *c.Configuration, supervisor cluster.SupervisorHandler, clusterDao dao.ClusterDao, scheduleDAO dao.ScheduleDao, monitoring *monitoring.Monitoring) *Service {
	return &Service{
		Config:      config,
		supervisor:  supervisor,
		clusterDao:  clusterDao,
		scheduleDao: scheduleDAO,
		Monitoring:  monitoring,
	}
}

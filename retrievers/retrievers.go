package retrievers

// TODO: change name to retrieverimpl
import (
	"github.com/myntra/goscheduler/conf"
	"github.com/myntra/goscheduler/dao"
	p "github.com/myntra/goscheduler/monitoring"
	"github.com/myntra/goscheduler/retrieveriface"
)

// Default app name
const _default = "<<DEFAULT>>"

// Type maps an app name to the corresponding schedule retriever.
type Retrievers map[string]retrieveriface.Retriever

// For a given app name, return the schedule retriever to be used.
func (retriever Retrievers) Get(app string) retrieveriface.Retriever {
	if output, ok := retriever[app]; ok {
		return output
	}

	return retriever[_default]
}

func InitRetrievers(cronConfig *conf.CronConfig, clusterDao dao.ClusterDao, scheduleDao dao.ScheduleDao, monitoring *p.Monitoring) Retrievers {
	cronApp := cronConfig.App
	return Retrievers{
		_default: ScheduleRetriever{clusterDao: clusterDao, scheduleDao: scheduleDao, monitoring: monitoring},
		cronApp:  CronRetriever{scheduleDao: scheduleDao, cronConfig: cronConfig, monitoring: monitoring},
	}
}

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

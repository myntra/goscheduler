package monitoring

import (
	gstatsd "github.com/cactus/go-statsd-client/statsd"
	"github.com/golang/glog"
	"github.com/myntra/goscheduler/conf"
	"github.com/uber-common/bark"
	statsd "gopkg.in/alexcesaro/statsd.v2"
)

func initStatsd(conf *conf.StatsdConfig) (*statsd.Client, error) {
	if conf.Enabled {
		statsdClient, err := statsd.New(
			statsd.Address(conf.Address),
			statsd.Prefix(conf.Prefix),
		)
		if err != nil {
			glog.Fatalf("Error while creating statsd client %+v", err)
			return statsdClient, err
		}
		return statsdClient, nil
	}

	return nil, nil

}

func GetRingPopStatsD(conf *conf.StatsdConfig) bark.StatsReporter {
	RingPopStatsDClient, err := gstatsd.New(conf.Address, conf.Prefix)
	if err != nil {
		glog.Errorf("Error while creating statsd client %+v", err)
		return nil
	}
	return bark.NewStatsReporterFromCactus(RingPopStatsDClient)
}

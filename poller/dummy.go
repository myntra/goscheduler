package poller

type Dummy struct {
	AppName               string
	PartitionId           int
}

func (d Dummy) Init() error {
	return nil
}

func (d Dummy) Start() {
}

func (d Dummy) Stop() {
}
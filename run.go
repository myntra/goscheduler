package main

import (
	"fmt"
	"github.com/myntra/goscheduler/conf"
	"github.com/myntra/goscheduler/scheduler"
	"github.com/myntra/goscheduler/store"
)

func main() {
	fmt.Println("Welcome to goscheduler [Myntra Scheduler Service].")
	//Load all the configs
	config := conf.InitConfig()
	s := scheduler.New(config, map[string]store.Factory{})
	s.Supervisor.WaitForTermination()
}

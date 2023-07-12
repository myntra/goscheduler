#!/bin/bash

# Start goscheduler on port 8080
echo "goscheduler service started on port 8080"
echo "going to sleep"
sleep 60
echo "waking up"
PORT=8080 ./goscheduler -h service1 -p 9091 -conf=./conf/conf.docker.json
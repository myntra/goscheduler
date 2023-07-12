#!/bin/bash

echo "Starting goscheduler on port 8081"
echo "going to sleep"
sleep 60
echo "waking up"
PORT=8081 ./goscheduler -h service2 -p 9091 -conf=./conf/conf.docker.json
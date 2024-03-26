# Table of Contents
1. [Introduction](#introduction)
2. [Architecture](#architecture)
    - [Tech Stack](#tech-stack)
    - [Service Layer](#service-layer)
    - [Datastore](#datastore)
    - [Poller Cluster](#poller-cluster)
        - [Poller Distribution](#poller-distribution)
        - [Scalability and Fault Tolerance](#scalability-and-fault-tolerance)
3. [How does it work?](#how-does-it-work)
4. [Getting Started](#getting-started)
    - [Installation](#installation)
        - [Approach 1: Using Docker](#approach-1-using-docker)
        - [Approach 2: Manual Setup](#approach-2-manual-setup)
        - [Unit Tests](#unit-tests)
    - [Configuration](#configuration)
5. [Usage](#usage)
    - [Use as Separate Service](#use-as-separate-service)
        - [Client Onboarding](#client-onboarding)
        - [Schedule Creation](#schedule-creation)
        - [Check Schedule Status](#check-schedule-status)
    - [Use as Go Module](#use-as-go-module)
        - [Client Onboarding (Go Module)](#client-onboarding-go-module)
        - [Create One Time Schedule (Go Module)](#create-one-time-schedule-go-module)
        - [Check Schedule Status (Go Module)](#check-schedule-status-go-module) 
6. [Use Cases](#use-cases)
7. [APIs](#apis)
8. [Benchmarks](#benchmarks)
9. [License](#license)
10. [Contributors](#contributors)
 
# Introduction
GoScheduler, a distributed scheduling platform based on Myntra's Scheduler Service ([MySS](https://medium.com/myntra-engineering/myntra-scheduler-service-a0153a04526c)), is an open-source project designed to handle high throughput with low latency for scheduled job executions. GoScheduler is based on [Uber Ringpop](https://github.com/uber/ringpop-go) and offers capabilities such as multi-tenancy, per-minute granularity, horizontal scalability, fault tolerance, and other essential features. GoScheduler is written in Golang and utilizes Cassandra DB, allowing it to handle high levels of concurrent create/delete and callback throughputs.

# Architecture
![Go Scheduler Architecture](./docs/images/go_scheduler_arch.png)

The Go Scheduler service consists of three major components - http service layer, poller cluster and datastore.

## Tech Stack
1. **Golang**: The service layer and poller layer of the Go Scheduler service are implemented using the Go programming language (Golang). Golang offers high throughput, low latency, and concurrency capabilities through its lightweight goroutines. It is well-suited for services that require efficient memory utilization and high concurrency.

2. **Cassandra**: Cassandra is chosen as the datastore for the Go Scheduler service. Cassandra offers horizontal scalability, fault tolerance, and distributed data handling capabilities. Its adoption by Myntra underscores its proficiency in managing scenarios with high write throughput, which is a critical requirement for GoScheduler, especially considering the major use case revolves around schedule creation.
   
## Service Layer
The service layer in the Scheduler service handles all REST traffic. It provides a web interface and exposes various endpoints for interacting with the service. Some of the important endpoints include:

- Register Client: This endpoint allows administrators to register a new client. A client represents a tenant, which is another service running its use case on the Go Scheduler service. Each client is registered with a unique ID.
- Schedule Endpoints: The service layer includes endpoints for creating schedules, cancelling schedules, checking status of the schedules, reconciling schedules etc. These endpoints are accessible only to registered clients.

## Datastore
The Scheduler service utilizes Cassandra as the datastore. It stores the following types of data:

- Schedule State Data: This includes the payload, callback details, and success/failure status after the trigger.
- Client Configuration Metadata: The datastore holds metadata related to client configurations.
- Poller Instance Statuses and Poller Node Membership: The status and membership information of poller instances are stored in the datastore.

More details on Cassandra can be found [here](Link to Github wiki)

## Poller Cluster
The Poller Cluster in the Scheduler service utilizes the [Uber ringpop-go library](https://github.com/uber/ringpop-go) for its implementation. Ringpop provides application-level sharding, creating a consistent hash ring of available Poller Cluster nodes. The ring ensures that keys are distributed across the ring, with specific parts of the ring owned by individual Poller Cluster nodes.

### Poller Distribution
Every client within the Scheduler service owns a fixed number of Poller instances. Let's consider the total number of Poller instances assigned to all clients across all nodes as X. If there are Y clients where each client owns C1x, C2x, ..., CYx Poller instances respectively (where C1x + C2x + ... + CYx = X), and there are N Poller Cluster nodes, then each node would run approximately X/N Poller instances (i.e., X/N = C1x/N + C2x/N + ... + CYx/N).

### Scalability and Fault Tolerance
The Poller Cluster exhibits scalability and fault tolerance characteristics. When a node goes down, X/N Poller instances automatically shift to the available N-1 nodes, maintaining the distribution across the remaining nodes. Similarly, when a new node is added to the cluster, X/(N+1) Poller instances are shifted to the new node, while each existing node gives away X/N - X/(N+1) Poller instances.

This approach ensures load balancing and fault tolerance within the Poller Cluster, enabling efficient task execution and distribution across the available nodes.

# How does it work?
The GoScheduler follows a specific workflow to handle client registrations and schedule executions:

- **Client Registration:** Clients register with a unique client ID and specify their desired poller instance quota. The poller instance quota is determined based on the client's callback throughput requirements.
- **Poller Instances:** Each poller instance fires every minute and is responsible for fetching schedules from the datastore. Each poller instance can fetch a maximum of 1000 schedules, with each schedule having a maximum payload size of 1KB.
Assigning Poller Instances: When a client registers, they are assigned a specific number of poller instances. For example, if a client requires a callback requirement of 50000 RPM, they might be assigned 50 (50+x, where x is a buffer for safety) poller instances. These poller instances are identified by the client ID followed by a numeric identifier (e.g., C1.1, C1.2, ..., C1.50).
- **Scheduling and Distribution:** When a client creates a schedule, it is tied to one of the assigned poller instances using a random function that ensures a uniform distribution across all poller instance IDs. For example, if 50000 schedules are created with a fire time of 5:00 PM, each poller instance for this client will be assigned approximately 1000 schedules to be triggered at 5:00 PM. The schedules tied to each poller instance are fetched and triggered at the respective callback channels.
- **Scaling:** The GoScheduler can be horizontally scaled based on the increasing throughput requirements. For higher create/delete peak RPM, additional service nodes or datastore nodes (or both) can be added. Similarly, for higher peak callback RPM, the number of poller instances for a client can be increased, which may require adding new nodes in the poller cluster or datastore (or both). This scalability ensures that the service can handle increasing throughput by augmenting nodes in the service layer, poller cluster, and datastore.


# Getting Started

## Installation

### Approach 1: Using Docker

1. Install [Docker](https://docs.docker.com/get-docker/) on your machine.
2. Clone the repository. 
2. Change the current directory to the repository directory: `cd ./goscheduler`.
3. Build and run the Docker containers using the following command: 
```shell
docker-compose --no-cache build
docker-compose up -d
```
This starts the service instances on ports 8080 and 8081, respectively, and the Ringpop instances on ports 9091 and 9092.

### Approach 2: Manual Setup

1. Install [Go](https://go.dev/dl/) (>= 1.17)
2. Install [Cassandra](https://cassandra.apache.org/_/download.html) (>= 3.0.0) on your machine.
3. Set the environment variables:
  - `GOROOT`: Set it to the directory path of the Go SDK.
  - `GOPATH`: Set it to the path of the directory where you want to store your Go workspace.
These environment variables are required for the Go toolchain to work correctly and to manage Go packages.
3. Run the following command in the repository directory to download and manage the project's dependencies:
```
go mod tidy
```
4. Build the service by running the following command in the repository directory:
```
go build .
```
5. Start multiple instances of service using following commands:
```shell
PORT=8080 ./goscheduler -h 127.0.0.1 -p 9091 -conf=./conf/conf.json
PORT=8081 ./goscheduler -h 127.0.0.1 -p 9092 -conf=./conf/conf.json
```
This starts the service instances on ports 8080 and 8081, respectively, and the Ringpop instances on ports 9091 and 9092.

### Unit tests
To run unit tests for go scheduler, you can use the following commands:
```
go test -v -coverpkg=./... -coverprofile=profile.cov ./...
go tool cover -func profile.cov
```

## Configuration
To configure the `conf.json` use the following guidelines:
```yml
{
  "HttpPort": "8080", # Port for HTTP communication
  "Cluster": {
    "Address": "127.0.0.1:9091", # Ringpop cluster address with IP and port
    "TChannelPort": "9091", # Ringpop port for TChannel communication
    "BootStrapServers": [
      "127.0.0.1:9091", "127.0.0.1:9092" # Ringpop cluster bootstrap server IPs and ports
    ]
    # ... other configurations ...
  },
  "ClusterDB": {
    "DBConfig": {
      "Hosts": "127.0.0.1", # Cassandra database host IP
      "VaultConfig": {
        "Enabled": false,
        "Username": "cassandra", # Cassandra database username if vaultConfig is enabled
        "Password": "cassandra" # Cassandra database password if vaultConfig is enabled
      }
      # ... other configurations ...
    }
    # ... other configurations ...
  },
  "ScheduleDB": {
    "DBConfig": {
      "Hosts": "127.0.0.1", # Cassandra database host IP
      "VaultConfig": {
        "Enabled": false,
        "Username": "cassandra", # Cassandra database username if vaultConfig is enabled
        "Password": "cassandra" # Cassandra database password if vaultConfig is enabled
      }
      # ... other configurations ...
    }
    # ... other configurations ...
  },
  # ... other configurations ...
  "MonitoringConfig": {
    "Statsd": {
      "Address": "54.251.41.202:8125" # Monitoring server IP and port
      # ... other configurations ...
    }
  },
  # ... other configurations ...
}
```

- `HttpPort`: Port for HTTP communication, e.g., `"8080"`
- `Cluster.Address`: Ringpop address with IP and port, e.g., `"127.0.0.1:9091"`
- `Cluster.TChannelPort`: Port for Ringpop TChannel communication, e.g., `"9091"`
- `Cluster.BootStrapServers`: Ringpop cluster bootstrap nodes, e.g., `["127.0.0.1:9091", "127.0.0.1:9092"]`
- `ClusterDB.DBConfig.Hosts`: Database host IP, e.g., `"127.0.0.1"`
- `ClusterDB.DBConfig.VaultConfig`: If enabled, provide username and password e.g., `"Username": "cassandra"` and `"Password": "cassandra"` 
- `ScheduleDB.DBConfig.Hosts`: Database host IP, e.g., `"127.0.0.1"`
- `ScheduleDB.DBConfig.VaultConfig`: If enabled, provide username and password e.g., `"Username": "cassandra"` and `"Password": "cassandra"`
- `MonitoringConfig.Statsd.Address`: Monitoring server IP and port, e.g., `"54.251.41.202:8125"`

To configure the service during startup, you can use the following options:

- `PORT`: Specify the port number for the service to listen on. For example, `PORT=8080`.

- `-h`: Set the hostname or IP address for the service. For example, `-h 127.0.0.1`.

- `-p`: Specify the port number for the Ringpop instance. For example, `-p 9091`.

- `-conf`: Provide the absolute path of a custom configuration file for the service. For example, `-conf /path/to/myconfig.json`.

- `-r`: Specify the port number where Ringpop is run for rate-limiting purposes. For example, `-r 2479`.

# Usage
Go Scheduler can be used as a separate service or as part of a Go module. Here's how you can incorporate Go Scheduler into your project:

## Use as Separate Service

### Client Onboarding
For any schedule creation, you need to register the app associated with it first. Additionally, when creating Cron Schedules, you need to register the **Athena** app (default app, which can be changed from the configuration).
Use the following API to create an app:

```bash
curl --location 'http://localhost:8080/goscheduler/app' \
--header 'Content-Type: application/json' \
--data '{
    "appId": "test",
    "partitions": 5,
    "active": true
}'
```

The request body should be a JSON object with the following fields:
- `appId (string)`: The ID of the app to create.
- `partitions (integer)`: The number of partitions for the app.
- `active (boolean)`: Specifies if the app is active or not.

The API will respond with the created app's details in JSON format.

```json
{
    "status": {
        "statusCode": 201,
        "statusMessage": "Success",
        "statusType": "Success",
        "totalCount": 1
    },
    "data": {
        "appId": "test",
        "partitions": 5,
        "active": true,
        "configuration": {}
    }
}
```

### Schedule Creation
#### Create One Time Schedule
```bash
curl --location 'http://localhost:8080/goscheduler/schedule' \
--header 'Content-Type: application/json' \
--data '{
    "appId": "test",
    "payload": "{}",
    "scheduleTime": 1686676947,
    "callback": {
        "type": "http",
        "details": {
            "url": "http://127.0.0.1:8080/goscheduler/healthcheck",
            "method": "GET",
            "headers": {
                "Content-Type": "application/json",
                "Accept": "application/json"
            }
        }
    }
}'
```

The request body should be a JSON object with the following fields:

- `appId (string)`: The ID of the app for which the schedule is created.
- `payload (string)`: The payload or data associated with the schedule. It can be an empty string or any valid JSON data.
- `scheduleTime (integer)`: The timestamp representing the schedule time.
- `callback (object)`: The callback configuration for the schedule.
  - `type (string)`: The type of callback. In this example, it is set to "http".
  - `details (object)`: The details specific to the callback type. For the "http" callback, it includes the URL, HTTP method, and headers.


The API will respond with the created schedule's details in JSON format.

Example response body:
```json
{
    "status": {
        "statusCode": 201,
        "statusMessage": "Success",
        "statusType": "Success",
        "totalCount": 1
    },
    "data": {
        "scheduleId": "2358e5b6-09f3-11ee-a704-acde48001122",
        "appId": "test",
        "payload": "{}",
        "scheduleTime": 1686676947,
        "callback": {
            "type": "http",
            "details": {
                "url": "http://127.0.0.1:8080/goscheduler/healthcheck",
                "method": "GET",
                "headers": {
                    "Content-Type": "application/json",
                    "Accept": "application/json"
                }
            }
        }
    }
}
```

### Check Schedule Status
```
curl --location 'http://localhost:8080/goscheduler/schedule/a675115c-0a0e-11ee-bebb-acde48001122' \
--header 'Accept: application/json'
```

`{scheduleId}` is the actual UUID of the schedule you want to retrieve.

Example response body:
```json
{
    "status": {
        "statusCode": 200,
        "statusMessage": "Success",
        "statusType": "Success",
        "totalCount": 1
    },
    "data": {
        "schedule": {
            "scheduleId": "a675115c-0a0e-11ee-bebb-acde48001122",
            "payload": "{}",
            "appId": "test",
            "scheduleTime": 1686676947,
            "partitionId": 4,
            "scheduleGroup": 1686676920,
            "callback": {
                "type": "http",
                "details": {
                    "url": "http://127.0.0.1:8080/goscheduler/healthcheck",
                    "method": "GET",
                    "headers": {
                        "Accept": "application/json",
                        "Content-Type": "application/json"
                    }
                }
            },
            "status": "SUCCESS"
        }
    }
}
```

More details on APIs and Customisable callbacks can be found [here](https://github.com/myntra/goscheduler/wiki/APIs)

## Use as go module
If the application is in Golang, Go Scheduler can be used as a module directly instead of deploying it as a separate process.

### Client Onboarding (Go Module)

```go
package main

import (
	"fmt"
	"time"
	sch "github.com/myntra/goscheduler/scheduler"
	"github.com/myntra/goscheduler/store"
)

func main() {
	// Create a Scheduler instance using a configuration loaded from a file
	scheduler := sch.FromConfFile("config.json")
	service := scheduler.Service

	// Register App
	registerAppPayload := store.App{
		AppId:      "my-app",
		Partitions: 4,
		Active:     true,
	}

	registeredApp, err := service.RegisterApp(registerAppPayload)
	if err != nil {
		fmt.Printf("Failed to register app: %v\n", err)
		return
	}
	fmt.Printf("Registered app: %+v\n", registeredApp)
 }
```

### Create One Time Schedule (Go Module)

```go
package main

import (
	"fmt"
	"time"
	sch "github.com/myntra/goscheduler/scheduler"
	"github.com/myntra/goscheduler/store"
)

func main() {
	// Create a Scheduler instance using a configuration loaded from a file
	scheduler := sch.FromConfFile("config.json")
	service := scheduler.Service

	// Create a Schedule with a sample HTTP Callback
	createSchedulePayload := store.Schedule{
		AppId:        "test",
		Payload:      "{}",
		ScheduleTime: time.Now().Unix(),
		Callback: store.Callback{
			Type: "http",
			Details: store.HTTPCallback{
				Url: "http://127.0.0.1:8080/test/healthcheck",
				Method: "GET",
				Headers: map[string]string{
					"Content-Type": "application/json",
					"Accept":       "application/json",
				},
			},
		},
	}

	createdSchedule, err := service.CreateSchedule(createSchedulePayload)
	if err != nil {
		fmt.Printf("Failed to create schedule: %v\n", err)
		return
	}
	fmt.Printf("Created schedule: %+v\n", createdSchedule)
 }
```

### Check Schedule Status (Go Module)

```go
package main

import (
	"fmt"
	"time"
	sch "github.com/myntra/goscheduler/scheduler"
	"github.com/myntra/goscheduler/store"
)

func main() {
	// Create a Scheduler instance using a configuration loaded from a file
	scheduler := sch.FromConfFile("config.json")
	service := scheduler.Service

	// Get Schedule
	scheduleUUID := "f47ac10b-58cc-4372-a567-0e02b2c3d479"

	schedule, err := service.GetSchedule(scheduleUUID)
	if err != nil {
		fmt.Printf("Failed to get schedule: %v\n", err)
		return
	}
	fmt.Printf("Retrieved schedule: %+v\n", schedule)
 }
```
More details on go module integration can be found [here](https://github.com/myntra/goscheduler/wiki/Use-as-Go-module)


# Use Cases
In general, goscheduler can be used to schedule jobs with customizable callbacks at scale. Some of the real-world use-cases are as follows
- **Task Scheduling:** Schedule tasks or jobs to run at specific times or intervals, allowing for automated execution of recurring or time-sensitive operations.

- **Event Triggering:** Schedule events to be triggered based on specific conditions or external triggers, enabling event-driven architectures and workflows.

- **Reminder Services:** Create schedules for sending reminders or notifications to users for appointments, deadlines, or important events.

- **Service Level Agreements (SLAs):** Schedule SLA checks for different stages in a workflow or business process, ensuring that tasks or activities are completed within predefined time constraints. If an SLA breach occurs, schedules can be triggered to take appropriate actions or notify stakeholders.

- **Retries and Retry Strategies:** Handle failures or errors in asynchronous processing by scheduling retries with backoff strategies. The scheduler can automatically schedule retries based on configurable policies, allowing for resilient and fault-tolerant processing.

- **Payment Reconciliation:** Schedule reconciliation tasks for payment processing systems to ensure the consistency and accuracy of transactions. For example, if a payment gateway experiences issues or timeouts, the scheduler can schedule a reconciliation task to fetch transaction status from the bank and initiate necessary actions like refunds.

More details on usecases can be found [here](https://github.com/myntra/goscheduler/wiki/Use-Cases)

# APIs
Detailed API documentation for goscheduler can be found [here](https://github.com/myntra/goscheduler/wiki/APIs)

# Benchmarks

| Scenario                                                                                               | RPM          | Duration   | Latency   |
|--------------------------------------------------------------------------------------------------------|--------------|------------|-----------|
| Create Schedule with async Kafka-based callback running simultaneously                                 | 350K         | 20 min     | p99 < 50ms|
| Create Schedule (240K), async Kafka-based callback, delete schedule (60K) running simultaneously       | 300K         | 40 min     | p99 < 60ms|
| Create Schedule with HTTP callback running simultaneously                                              | 100K         | 15 min     | p99 < 30ms|

**Note:** All the runs are made with following configurations: 8 application servers with [Standard_D8_v3 Azure boxes](https://learn.microsoft.com/en-us/azure/virtual-machines/dv3-dsv3-series), 7 node Cassandra cluster with [Standard_D16_v3 Azure boxes](https://learn.microsoft.com/en-us/azure/virtual-machines/dv3-dsv3-series)


# License
This project is licensed under the MIT License - see the [LICENSE.md](LICENSE.md) file for details

# Contributors
A big thank you to everyone who has contributed to this project!
  
<!-- Use https://contrib.rocks/preview?repo=myntra%2Fgoscheduler-->

<!-- <a href = "https://github.com/Tanu-N-Prabhu/Python/graphs/contributors">
  <img src = "https://contrib.rocks/image?repo = myntra/goscheduler"/>
</a>

Made with [contributors-img](https://contrib.rocks).
-->

If you'd like to contribute, please see the [Contributing](CONTRIBUTING.md) guide.

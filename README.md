# Table of Contents
1. [Introduction](#introduction)
2. [Architecture](#architecture)
    - [Tech Stack](#tech-stack)
    - [Service Layer](#service-layer)
    - [Datastore](#datastore)
      - [Cassandra Data Model](#cassandra-data-model)   
    - [Poller Cluster](#poller-cluster)
      - [Poller Distribution](#poller-distribution)
      - [Scalability and Fault Tolerance](#scalability-and-fault-tolerance)
3. [How does it work?](#how-does-it-work) 
4. [Getting Started](#getting-started)
    - [Installation](#installation)
      - [Approach 1: Using Docker](#approach-1-using-docker)
      - [Approach 2: Manual Setup](#approach-2-manual-setup)
    - [Configuration](#configuration)
    - [Use as separate service](#use-as-separate-service)
      - [Client onboarding](#client-onboarding)
      - [Schedule creation](#schedule-creation)
        - [Create one-time schedule](#create-one-time-schedule)
        - [Create cron schedule](#create-cron-schedule)
      - [Check Schedule Status](#check-schedule-status)
      - [Customizable Callback](#customizable-callback)
    - [Use as go module](#use-as-go-module)
      - [Register App](#register-app)
      - [Create One Time Schedule](#create-one-time-schedule)
      - [Create Cron Schedule](#create-cron-schedule)
      - [Get Schedule](#get-schedule)
      - [Customised Callback](#customizable-callback)
 
# Introduction
GoScheduler, also known as Myntra's Scheduler Service (MySS), is an open-source project designed to handle high throughput with low latency for scheduled job executions. GoScheduler is based on [Uber Ringpop](https://github.com/uber/ringpop-go) and offers capabilities such as multi-tenancy, per-minute granularity, horizontal scalability, fault tolerance, and other essential features. GoScheduler is written in Golang and utilizes Cassandra DB, allowing it to handle high levels of concurrent create/delete and callback throughputs. Further information about GoScheduler can be found in this [article](https://medium.com/myntra-engineering/myntra-scheduler-service-a0153a04526c).

# Architecture
![Go Scheduler Architecture](./docs/images/go_scheduler_arch.png)

The Go Scheduler service consists of three major components - http service layer, poller cluster and datastore.

## Tech Stack
1. **Golang**: The service layer and poller layer of the Go Scheduler service are implemented using the Go programming language (Golang). Golang offers high throughput, low latency, and concurrency capabilities through its lightweight goroutines. It is well-suited for services that require efficient memory utilization and high concurrency.

2. **Cassandra**: Cassandra is chosen as the datastore for the Go Scheduler service. It provides horizontal scalability, fault tolerance, and distributed data storage. Cassandra is widely used within Myntra and is known for its ability to handle high write throughput scenarios.

## Service Layer
The service layer in the Scheduler service handles all REST traffic. It provides a web interface and exposes various endpoints for interacting with the service. Some of the important endpoints include:

- Register Client: This endpoint allows administrators to register a new client. A client represents a tenant, which is another service running its use case on the Go Scheduler service. Each client is registered with a unique ID.
- Schedule Endpoints: The service layer includes endpoints for creating schedules, cancelling schedules, checking status of the schedules, reconciling schedules etc. These endpoints are accessible only to registered clients.

## Datastore
The Scheduler service utilizes Cassandra as the datastore. It stores the following types of data:

- Schedule State Data: This includes the payload, callback details, and success/failure status after the trigger.
- Client Configuration Metadata: The datastore holds metadata related to client configurations.
- Poller Instance Statuses and Poller Node Membership: The status and membership information of poller instances are stored in the datastore.

### Cassandra Data Model
The data layout in Cassandra is designed in a way that during every minute, the poller instance request goes to a single Cassandra shard, ensuring fast reads.
The schedule table has the following primary key structure: (clientid, poller count id, schedule minute).

For example, the data for the use case mentioned earlier will look like:

```
C1, 1, 5:00PM, uuid1, payload
C1, 2, 5:00PM, uuid2, payload
...
C1, 50, 5:00PM, uuid50000, payload
```
Here, `uuid1` to `uuid50000` are unique schedule IDs.

In this data model, we perform 50,000 Cassandra writes and 50 Cassandra reads for the given use case.

It's important to note that not every schedule fire requires a read operation. The number of Cassandra reads in a minute is equal to the number of poller instances running for all clients. As Cassandra is well-suited for high write throughput and lower read throughput, this data modeling and poller design work effectively with the Cassandra datastore layer.

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
PORT=8080 ./myss -h 127.0.0.1 -p 9091
PORT=8081 ./myss -h 127.0.0.1 -p 9092
```
This starts the service instances on ports 8080 and 8081, respectively, and the Ringpop instances on ports 9091 and 9092.

## Configuration

To configure the service, you can use the following options:

- `PORT`: Specify the port number for the service to listen on. For example, `PORT=8080`.

- `-h`: Set the hostname or IP address for the service. For example, `-h 127.0.0.1`.

- `-p`: Specify the port number(s) for the Ringpop instances. For example, `-p 9091` or `-p 9091,9092`.

- `-conf`: Provide the absolute path of a custom configuration file for the service. For example, `-conf /path/to/myconfig.yaml`.

- `-r`: Specify the port number where Ringpop is run for rate-limiting purposes. For example, `-r 2479`.

## Use as separate service

### Client onboarding
For any schedule creation, you need to register the app associated with it first. Additionally, when creating Cron Schedules, you need to register the **Athena** app (default app, which can be changed from the configuration).
Use the following API to create an app:

```bash
curl --location 'http://localhost:8080/myss/app' \
--header 'Content-Type: application/json' \
--data '{
    "appId": "test",
    "partitions": 5,
    "active": true
}'
```

Request Body
The request body should be a JSON object with the following fields:
- `appId (string)`: The ID of the app to create.
- `partitions (integer)`: The number of partitions for the app.
- `active (boolean)`: Specifies if the app is active or not.

Response
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

### Schedule creation
#### Create one time schedule
```bash
curl --location 'http://localhost:8080/myss/schedule' \
--header 'Content-Type: application/json' \
--data '{
    "appId": "test",
    "payload": "{}",
    "scheduleTime": 1686676947,
    "callback": {
        "type": "http",
        "details": {
            "url": "http://127.0.0.1:8080/myss/healthcheck",
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
                "url": "http://127.0.0.1:8080/myss/healthcheck",
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
#### Create cron schedule
```bash
curl --location 'http://localhost:8080/myss/schedule' \
--header 'Content-Type: application/json' \
--data '{
    "appId": "test",
    "payload": "{}",
    "cronExpression": "*/5 * * * *",
    "callback": {
        "type": "http",
        "details": {
            "url": "http://127.0.0.1:8080/myss/healthcheck",
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

- `appId (string)`: The ID of the app for which the cron schedule is created.
- `payload (string)`: The payload or data associated with the cron schedule. It can be an empty string or any valid JSON data.
- `cronExpression (string)`: The cron expression representing the schedule pattern.
- `callback (object)`: The callback configuration for the schedule.
  - `type (string)`: The type of callback. In this example, it is set to "http".
  - `details (object)`: The details specific to the callback type. For the "http" callback, it includes the URL, HTTP method, and headers.

**Supported Cron Expression**

Go Scheduler supports standard UNIX cron expression of the following pattern.
```
* * * * *
| | | | |
| | | | └─ Day of the week (0 - 6, where 0 represents Sunday)
| | | └── Month (1 - 12)
| | └──── Day of the month (1 - 31)
| └────── Hour (0 - 23)
└──────── Minute (0 - 59)
```

The Cron Expression consists of five fields, each separated by a space:

| Field         | Required | Allowed Values         | Symbols     |
|---------------|----------|------------------------|-------------|
| Minutes       | Yes      | 0-59                   | * ,/-       |
| Hours         | Yes      | 0-23                   | * ,/-       |
| Day of month  | Yes      | 1-31                   | * ,/-       |
| Month         | Yes      | 1-12 or JAN-DEC        | * ,-        |
| Day of week   | Yes      | 0-6 or SUN-SAT         | * ,-        |

The supported symbols and their meanings are as follows:

- `*`: Matches all possible values.
- `,`: Specifies multiple values.
- `/`: Specifies stepping values.
- `-`: Specifies a range of values.

Examples
- `0 0 * * *`: Executes a task at midnight every day.
- `0 12 * * MON-FRI`: Executes a task at 12 PM (noon) from Monday to Friday.
- `*/15 * * * *`: Executes a task every 15 minutes.
- `0 8,12,18 * * *`: Executes a task at 8 AM, 12 PM (noon), and 6 PM every day.

The API will respond with the created cron schedule's details in JSON format.

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
        "schedule": {
            "scheduleId": "35b5e19e-0aa4-11ee-8563-acde48001122",
            "payload": "{}",
            "appId": "test",
            "partitionId": 3,
            "callback": {
                "type": "http",
                "details": {
                    "url": "http://127.0.0.1:8080/myss/healthcheck",
                    "method": "GET",
                    "headers": {
                        "Content-Type": "application/json",
                        "Accept": "application/json"
                    }
                }
            },
            "cronExpression": "*/5 * * * *",
            "status": "SCHEDULED"
        }
    }
}
```

### Check Schedule Status
```
curl --location 'http://localhost:8080/myss/schedule/a675115c-0a0e-11ee-bebb-acde48001122' \
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
                    "url": "http://127.0.0.1:8080/myss/healthcheck",
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
### Customizable Callback
GoScheduler's Callback feature is designed to be extensible, enabling users to define and utilize their own custom callbacks. The Callback structure serves as the foundation for creating customized callbacks tailored to specific requirements. By implementing the methods defined in the Callback interface, users can extend the functionality of the GoScheduler with their own callback implementations.

The Callback structure in GoScheduler consists of the following methods:
```go
type Callback interface {
	GetType() string
	GetDetails() (string, error)
	Marshal(map[string]interface{}) error
	Invoke(wrapper ScheduleWrapper) error
	Validate() error 
	json.Unmarshaler 
}
```

The methods in the Callback interface provide the necessary functionality for integrating customized callbacks into the GoScheduler.
- **GetType() string**: Returns the type or identifier of the callback.
- **GetDetails() (string, error)**: Retrieves the details of the callback, typically in JSON format.
- **Marshal(map[string]interface{}) error**: Deserializes the callback details from a map or JSON representation.
- **Invoke(wrapper ScheduleWrapper) error**: Executes the logic associated with the callback when it is triggered.
- **Validate() error**: Performs validation checks on the callback's details to ensure they are properly configured.

Sample Example
```go
type FooBarCallback struct {
	Type    string `json:"type"`
        Details string `json:"details"`
}

func (f *FooBarCallback) GetType() string {
	return f.Type
}

func (f *FooBarCallback) GetDetails() (string, error) {
	detailsJSON, err := json.Marshal(details)
	if err != nil {
		return "", err
	}
	return string(detailsJSON), nil
}

func (f *FooBarCallback) Marshal(m map[string]interface{}) error {
	// Sample implementation: Unmarshal the provided map to set the fields of FooBarCallback
	typeAlias := struct {
		Type string `json:"type"`
	}{}
	if err := mapstructure.Decode(m, &typeAlias); err != nil {
		return err
	}
	f.Type = typeAlias.Type
	return nil
}

func (f *FooBarCallback) Invoke(wrapper ScheduleWrapper) error {
    // Invoke the FooBar callback logic
    err := invokeFooBarCallback()

    if err != nil {
        wrapper.Schedule.Status = Failure
        wrapper.Schedule.ErrorMessage = err.Error()
    } else {
        wrapper.Schedule.Status = Success
    }

    // Push the updated schedule to the Aggregate Channel
    AggregateChannel <- wrapper

    return nil
}

func (f *FooBarCallback) Validate() error {
	// Sample implementation: Perform validation logic for FooBar callback
	if f.Type == "" {
		return errors.New("FooBar callback type is required")
	}

	// Additional validation logic specific to FooBar callback

	return nil
}
```
In this example, FooBarCallback is a custom callback implementation that defines the specific behavior for the Callback interface methods. Customize the fields and logic according to your requirements.

**Usage in startup file**

To incorporate the custom callback implementation in the GoScheduler, you need to make changes in the startup file (main function) as follows:

```go
func main() {
	// Load all the configs
	config := conf.InitConfig()

	// Create the custom callback factories map
	customCallbackFactories := map[string]store.Factory{
		"foo-bar": func() store.Callback { return &foobar.FooBarCallback{} },
		// Add more custom callback factories as needed
	}


	// Create the scheduler with the custom callback factories
	s := scheduler.New(config, customCallbackFactories)

	// Wait for the termination signal
	s.Supervisor.WaitForTermination()
}
```
## Use as go module
If the application is in Golang, Go Scheduler can be used as a module directly instead of deploying it as a separate process.
### Register App

```go
package main

import (
	"fmt"
	"time"
	sch "github.com/example/goScheduler/scheduler"
	"github.com/example/goScheduler/store"
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

### Create One Time Schedule

```go
package main

import (
	"fmt"
	"time"
	sch "github.com/example/goScheduler/scheduler"
	"github.com/example/goScheduler/store"
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

### Create Cron Schedule
Make sure to create Athena app before creating any Cron Schedules
```go
package main

import (
	"fmt"
	"time"
	sch "github.com/example/goScheduler/scheduler"
	"github.com/example/goScheduler/store"
)

func main() {
	// Create a Scheduler instance using a configuration loaded from a file
	scheduler := sch.FromConfFile("config.json")
	service := scheduler.Service

	// Create a Schedule with a sample HTTP Callback
	createSchedulePayload := store.Schedule{
		AppId:        "test",
		Payload:      "{}",
		CronExpression: "* * * * *",
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

### Get Schedule

```go
package main

import (
	"fmt"
	"time"
	sch "github.com/example/goScheduler/scheduler"
	"github.com/example/goScheduler/store"
)

func main() {
	// Create a Scheduler instance using a configuration loaded from a file
	scheduler := sch.FromConfFile("config.json")
	service := scheduler.Service

	// Get Schedule
	scheduleUUID := "12345"

	schedule, err := service.GetSchedule(scheduleUUID)
	if err != nil {
		fmt.Printf("Failed to get schedule: %v\n", err)
		return
	}
	fmt.Printf("Retrieved schedule: %+v\n", schedule)
 }
```

### Customised Callback
Using `FooBarCallback` defined earlier

```go
package main

import (
	"fmt"

	sch "github.com/myntra/goscheduler/scheduler"
	"github.com/myntra/goscheduler/store"
)

func main() {
	// Create a map of callback factories
	callbackFactories := map[string]sch.CallbackFactory{
		"foobar": func() store.Callback { return &FooBarCallback{} },
	}

	// Load the configuration file and create a Scheduler instance
	scheduler := sch.FromFileWithCallbacks(callbackFactories, "config.json")

	// Create a sample schedule payload
	schedule := store.Schedule{
		AppId:        "test",
		Payload:      "{}",
		ScheduleTime: 1686748449,
		Callback: FooBarCallback{
			Type:    "foobar",
			Details: "Custom details for FooBar callback",
		},
	}

	// Create the schedule using the CreateSchedule function
	createdSchedule, err := CreateSchedule(schedule)
	if err != nil {
		fmt.Println("Error creating schedule:", err)
		return
	}

	// Print the created schedule
	fmt.Println("Created schedule:", createdSchedule)
}
```


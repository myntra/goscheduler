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
        - [Unit Tests](#unit-tests)
    - [Configuration](#configuration)
5. [Usage](#usage)
    - [Use as Separate Service](#use-as-separate-service)
        - [Client Onboarding](#client-onboarding)
        - [Schedule Creation](#schedule-creation)
            - [Create One Time Schedule](#create-one-time-schedule)
            - [Create Cron Schedule](#create-cron-schedule)
        - [Check Schedule Status](#check-schedule-status)
        - [Customizable Callback](#customizable-callback)
    - [Use as Go Module](#use-as-go-module)
        - [Client Onboarding (Go Module)](#client-onboarding-go-module)
        - [Create One Time Schedule (Go Module)](#create-one-time-schedule-go-module)
        - [Create Cron Schedule (Go Module)](#create-cron-schedule-go-module)
        - [Check Schedule Status (Go Module)](#check-schedule-status-go-module)
        - [Customizable Callback (Go Module)](#customizable-callback-go-module)  
6. [Use Cases](#use-cases)
7. [API Contract](#api-contract)
 
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
PORT=8080 ./goscheduler -h 127.0.0.1 -p 9091
PORT=8081 ./goscheduler -h 127.0.0.1 -p 9092
```
This starts the service instances on ports 8080 and 8081, respectively, and the Ringpop instances on ports 9091 and 9092.

### Unit tests
To run unit tests for go scheduler, you can use the following commands:
```
go test -v -coverpkg=./... -coverprofile=profile.cov ./...
go tool cover -func profile.cov
```

## Configuration

To configure the service, you can use the following options:

- `PORT`: Specify the port number for the service to listen on. For example, `PORT=8080`.

- `-h`: Set the hostname or IP address for the service. For example, `-h 127.0.0.1`.

- `-p`: Specify the port number(s) for the Ringpop instances. For example, `-p 9091` or `-p 9091,9092`.

- `-conf`: Provide the absolute path of a custom configuration file for the service. For example, `-conf /path/to/myconfig.yaml`.

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
#### Create Cron Schedule
```bash
curl --location 'http://localhost:8080/goscheduler/schedule' \
--header 'Content-Type: application/json' \
--data '{
    "appId": "test",
    "payload": "{}",
    "cronExpression": "*/5 * * * *",
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
                    "url": "http://127.0.0.1:8080/goscheduler/healthcheck",
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
## Use as Go Module
If the application is in Golang, Go Scheduler can be used as a module directly instead of deploying it as a separate process.
### Client Onboarding (Go Module)

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

### Create One Time Schedule (Go Module)

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

### Create Cron Schedule (Go Module)
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

### Check Schedule Status (Go Module)

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

### Customizable Callback (Go Module)
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

# Use Cases
In general, goscheduler can be used to schedule jobs with customizable callbacks at scale. Some of the real-world use-cases are as follows
- **Task Scheduling:** Schedule tasks or jobs to run at specific times or intervals, allowing for automated execution of recurring or time-sensitive operations.

- **Event Triggering:** Schedule events to be triggered based on specific conditions or external triggers, enabling event-driven architectures and workflows.

- **Reminder Services:** Create schedules for sending reminders or notifications to users for appointments, deadlines, or important events.

- **Data Processing and ETL (Extract, Transform, Load):** Schedule data extraction, transformation, and loading processes to run at specified intervals or on-demand, facilitating data synchronization and integration.

- **Report Generation:** Schedule the generation and delivery of reports, enabling periodic or on-demand reporting for business intelligence or analytics purposes.

- **System Maintenance and Health Checks:** Schedule system maintenance tasks, such as database backups, log purging, or system health checks, to ensure smooth operations and proactive monitoring.

- **Resource Allocation and Optimization:** Schedule resource allocation and optimization processes, such as capacity planning, load balancing, or resource provisioning, to optimize resource utilization and performance.

- **Batch Processing:** Schedule batch processing tasks, such as data imports, batch updates, or batch computations, to efficiently process large volumes of data in scheduled batches.

- **Workflow Orchestration:** Schedule and orchestrate complex workflows or business processes involving multiple interconnected tasks or stages, ensuring the sequential or parallel execution of tasks based on predefined schedules.

- **Service Level Agreements (SLAs):** Schedule SLA checks for different stages in a workflow or business process, ensuring that tasks or activities are completed within predefined time constraints. If an SLA breach occurs, schedules can be triggered to take appropriate actions or notify stakeholders.

- **Retries and Retry Strategies:** Handle failures or errors in asynchronous processing by scheduling retries with backoff strategies. The scheduler can automatically schedule retries based on configurable policies, allowing for resilient and fault-tolerant processing.

- **Payment Reconciliation:** Schedule reconciliation tasks for payment processing systems to ensure the consistency and accuracy of transactions. For example, if a payment gateway experiences issues or timeouts, the scheduler can schedule a reconciliation task to fetch transaction status from the bank and initiate necessary actions like refunds.

# GoScheduler API Contract

Common Header for all the below APIs:

### Headers

|Header-Type|Value|
|---|---|
|Accept|application/json|
|Content-Type|application/json


## 1. HealthCheck API
This API is used to perform a health check or status check for the server.It checks whether the service is up and running.

This function provides a simple health check endpoint for the server, indicates the server's status to the client.

### Method: GET
```
http://localhost:8080/goscheduler/healthcheck
```

### Curl

```bash
curl --location --request GET 'http://localhost:8080/goscheduler/healthcheck'
```

### Sample Success Response: 200
``` json
{
    "statusMessage": "Success"
}
```
### Sample Error Response: 404
```json
{
    "statusMessage": "Fail"
}
```

## 2. Register App
This API handles the registration of an application. It receives a JSON payload containing the application information, inserts the application and its entities into the database, and returns an appropriate response indicating the registration status.


1. Check if the `AppId` field in the `payload` is empty. If it is empty, it records the registration failure and returns an appropriate response indicating that the AppId cannot be empty.

2. If the `Partitions` field in the `payload` is zero, it assigns the default count from the service's configuration.
3. If the `active` parameter is "TRUE", it represents that it is an active app and if it is "FALSE", it represents a deactivated app.

### Method: POST
```
http://localhost:8080/goscheduler/apps
```
### Body (**raw**)

```json
{
	"appId": "revamp",
	"partitions": 2,
        "active": true
}
```
### Curl

```bash
curl --location --request POST 'http://localhost:8080/goscheduler/apps' \
--header 'Content-Type: application/json' \
--data-raw '{
	"appId": "athena",
	"partitions": 2,
    "active": true
}'
```

### Sample Success Response: 200
```json
{
    "status": {
        "statusCode": 201,
        "statusMessage": "Success",
        "statusType": "Success",
        "totalCount": 1
    },
    "data": {
        "appId": "revamp",
        "partitions": 2,
        "active": true
    }
}
```

### Sample Error Response: 400
```json
{
    "status": {
        "statusCode": 400,
        "statusMessage": "AppId cannot be empty",
        "statusType": "Fail"
    }
}
```

## 3. Get Apps API
This API retrieves information about multiple apps based on "app_id" query parameter. If there is no "app_id" present in the request, it retrieves all the apps with its status.

If the status of "active" parameter is "TRUE", it represents that it is an active app and if it is "FALSE", it represents a deactivated app.
### Method: GET
```
http://localhost:8080/goscheduler/apps
```

If we want to retrieve information about a specific app, we can add the "app_id" parameter as a query param.


### Query Params

|Param| Description                                             | Type of Value |Sample value|
|---|---------------------------------------------------------|--------|---|
|app_id| The ID of the app for which the schedule is created | String | revamp |


### curl

```bash
curl --location --request GET 'http://localhost:8080/goscheduler/apps?app_id=revamp' \
--header 'Accept: application/json' \
--header 'Authorization: Basic ZXJwYWRtaW46d2VsY29tZUAyNTg=' \
--data-raw ''
```

### Sample Success Response: 200
```json
{
    "status": {
        "statusCode": 200,
        "statusMessage": "Success",
        "statusType": "Success",
        "totalCount": 2
    },
    "data": {
        "apps": [
            {
                "appId": "opensource",
                "partitions": 5,
                "active": true
            },
            {
                "appId": "revamp",
                "partitions": 2,
                "active": true
            }
        ]
    }
}
```

## 4. Create Schedule HttpCallback API
This API creates a schedule based on the data provided in the request body.

Based on the appId present in the payload of the API, it handles different error scenarios.

1. **If the app is not found** - It records a create failure, handles the **error indicating an invalid app ID**, and returns an appropriate response.
2. If there is **an error while fetching the app -** it records a create failure, **handles the data fetch failure error.**
3. If the **app ID is empty -** it records a create failure, handles the **error indicating an invalid app ID.**
4. If the **app is not active -** it records a create failure, handles **the error indicating a deactivated app.**

### Method: POST
```
http://localhost:8080/goscheduler/schedules
```

### Body (**raw**)

```json
{
    "appId": "revamp",
    "payload": "{}",
    "scheduleTime": 2687947561,
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
```
### Curl

```bash
curl --location --request POST 'http://localhost:8080/goscheduler/schedules' \
--data-raw '{
    "appId": "revamp",
    "payload": "{}",
    "scheduleTime": 2687947561,
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

### Sample Success Response: 200

```
{
    "status": {
        "statusCode": 201,
        "statusMessage": "Success",
        "statusType": "Success",
        "totalCount": 1
    },
    "data": {
        "schedule": {
            "scheduleId": "1497b35c-1a21-11ee-8689-ceaebc99522c",
            "payload": "{}",
            "appId": "revamp",
            "scheduleTime": 2529772970,
            "Ttl": 0,
            "partitionId": 1,
            "scheduleGroup": 2529772920,
            "httpCallback": {
                "url": "http://127.0.0.1:8080/goscheduler/healthcheck",
                "method": "GET",
                "headers": {
                    "Accept": "application/json",
                    "Content-Type": "application/json"
                }
            }
        }
    }
}
```


### Sample Error Response: 400
```json
{
    "status": {
        "statusCode": 400,
        "statusMessage": "parse \"www.myntra.com\": invalid URI for request,Invalid http callback method ,schedule time : 1629772970 is less than current time: 1688443428 for app: revamp. Time cannot be in past.",
        "statusType": "Fail"
    }
}
```

## 5. Create Cron-Schedule API

The purpose of this API is to create a cron schedule based on the cron expression provided in the request body.

To create a cron schedule, you will first have to register the **Athena** app.

### Method: POST
```
http://localhost:8080/goscheduler/schedules
```
### Body (**raw**)

```json
{
    "appId": "athena",
    "payload": "{}",
    "cronExpression": "*/5 * * * *",
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
```

### curl
```bash
curl --location --request POST 'http://localhost:8080/goscheduler/schedules' \
--header 'Authorization: Basic ZXJwYWRtaW46d2VsY29tZUAyNTg=' \
--header 'Content-Type: application/json' \
--data-raw '{
    "appId": "Athena",
    "payload": "{}",
    "cronExpression": "*/5 * * * *",
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

### Sample Success Response: 200
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
            "scheduleId": "167233ef-1fce-11ee-ba66-0242ac120004",
            "payload": "{}",
            "appId": "Athena",
            "partitionId": 0,
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
            },
            "cronExpression": "*/6 * * * *",
            "status": "SCHEDULED"
        }
    }
}
```

### Sample Error Response: 400
```json
{
    "status": {
        "statusCode": 400,
        "statusMessage": "parse \"www.google.com\": invalid URI for request,Invalid http callback method ",
        "statusType": "Fail"
    }
}
```

## 6. Get Cron-Schedules API
This HTTP endpoint retrieves cron schedules based on the provided parameters - app_id and status.

The various values of "STATUS" can be -

|Status| Description |
|----|-------------|
|SCHEDULED|Represents the schedule is scheduled |
|DELETED| Represents the schedule is deleted |
|SUCCESS| Represents the schedule is successfully run | 
|FAILURE| Represents the schedule is failed |
|MISS| Represents the schedule was not triggered |

### Method: POST
```
http://localhost:8080/goscheduler/crons/schedules?app_id=revamp
```
### Query Params


|Param| Description                                              | Type of Value | Example         |
|---|----------------------------------------------------------|--------|-----------------|
|app_id| The ID of the app for which the cron schedule is created | string | revamp |

### curl
```bash
curl --location --request GET 'http://localhost:8080/goscheduler/crons/schedules?app_id=revamp&status=SCHEDULED' \
--header 'Authorization: Basic ZXJwYWRtaW46d2VsY29tZUAyNTg='
```


### Sample Success Response:
```json
{
  "status": {
    "statusCode": 200,
    "statusMessage": "Success",
    "statusType": "Success",
    "totalCount": 0
  },
  "data": [
    {
      "scheduleId": "167233ef-1fce-11ee-ba66-0242ac120004",
      "payload": "{}",
      "appId": "revamp",
      "partitionId": 0,
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
      "cronExpression": "*/6 * * * *",
      "status": "SCHEDULED"
    }
  ]
}
```
### Sample Error response
```json
{
    "status": {
        "statusCode": 404,
        "statusMessage": "No cron schedules found",
        "statusType": "Fail"
    }
}
```

## 7. Get Schedule API
This API retrieves a schedule by its ID, handles different retrieval scenarios, and returns the appropriate response with the retrieved schedule or error information.

### Method: GET
```
http://localhost:8080/goscheduler/schedules/1497b35c-1a21-11ee-8689-ceaebc99522c
```

### Curl

```bash
curl --location --request GET 'http://localhost:8080/goscheduler/schedules/1497b35c-1a21-11ee-8689-ceaebc99522c' \
--header 'Accept: application/json' \
--header 'Authorization: Basic ZXJwYWRtaW46d2VsY29tZUAyNTg='
```

### Sample Success Response: 200
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
            "scheduleId": "1497b35c-1a21-11ee-8689-ceaebc99522c",
            "payload": "{}",
            "appId": "revamp",
            "scheduleTime": 2529772970,
            "partitionId": 1,
            "scheduleGroup": 2529772920,
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
            "status": "SCHEDULED"
        }
    }
}
```

### Sample Error Response: 404
```json
{
    "status": {
        "statusCode": 404,
        "statusMessage": "Schedule with id: 4588e23e-ae06-11ec-bcba-acde48001122 not found",
        "statusType": "Fail"
    }
}
```

## 8. Get Schedules with all runs API
This API retrieves the runs of a schedule (execution instances) of a schedule based on the schedule's ID. 
It handles different retrieval scenarios, and returns the appropriate response with the retrieved runs or error information.

If no runs are found, it logs an info message, records the successful retrieval, handles the data not found error, and returns an appropriate response.

### Method: GET
```
http://localhost:8080/goscheduler/schedules/1497b35c-1a21-11ee-8689-ceaebc99522c/runs?when=future&size=1
```

### Query Params

|Param| Description                         | Type of Value | Example     |
|---|-------------------------------------|---------------|-------------|
|when| time-frame                          | string        | past/future |
|size| number of results we want to fetch| int| 1           |

### Curl
```bash
curl --location --request GET 'http://localhost:8080/goscheduler/schedules/1497b35c-1a21-11ee-8689-ceaebc99522c/runs?when=future&size=1' \
--header 'Accept: application/json' \
--header 'Authorization: Basic ZXJwYWRtaW46d2VsY29tZUAyNTg=' \
--data-raw ''
```

### Sample Success Response: 200
```json
{
    "status": {
        "statusCode": 200,
        "statusMessage": "Success",
        "statusType": "Success",
        "totalCount": 11
    },
    "data": {
        "schedules": [
            {
                "scheduleId": "26631a1b-1fd6-11ee-ba9c-0242ac120004",
                "payload": "{}",
                "appId": "revamp",
                "scheduleTime": 1689071760,
                "partitionId": 0,
                "scheduleGroup": 1689071760,
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
                "status": "SCHEDULED"
            },
            {
                "scheduleId": "4fda1178-1fd5-11ee-ba96-0242ac120004",
                "payload": "{}",
                "appId": "revamp",
                "scheduleTime": 1689071400,
                "partitionId": 0,
                "scheduleGroup": 1689071400,
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
                "status": "FAILURE",
                "errorMessage": "404 Not Found"
            },
          {
            "scheduleId": "2c0df520-1fce-11ee-ba68-0242ac120004",
            "payload": "{}",
            "appId": "revamp",
            "scheduleTime": 1689068160,
            "partitionId": 0,
            "scheduleGroup": 1689068160,
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
            "status": "MISS",
            "errorMessage": "Failed to make a callback"
          }
        ],
      "continuationToken": ""
    }
}
```

### Sample Error Response: 404
```json
{
    "status": {
        "statusCode": 404,
        "statusMessage": "No runs found for schedule with id 1305aec5-e233-11ea-97ed-000d3af279cb",
        "statusType": "Fail"
    }
}
```

## 9. Get Paginated Schedules by appId API
This API get all the schedules associated with a specific application ID based on time range and status in a paginated way.

It throws an error if the application Id is not registered, or if the application details are not fetched successfully.

It handles different scenarios based on the parsed query parameters:

1. If there is an error parsing the query parameters, it handles the invalid data error and returns an appropriate response.
2. If the `size` parameter is less than 0, it handles the invalid data error.
3. If the `endTime` of the `timeRange` is before the `startTime`, it handles the invalid data error.
4. If the time range is greater than a predefined number of days (30 days in this case), it handles the invalid data error and returns an appropriate response.
5. If the query parameters are successfully parsed, the function calls the `GetPaginatedSchedules` method of the `scheduleDao` to retrieve paginated schedules based on the application ID and other parameters.


### Method: GET
```
http://localhost:8080/goscheduler/apps/revamp/schedules
```
### Query Params


| Param | Description                                    | Type of Value | Example         |
|-------|------------------------------------------------|---------------|-----------------|
| size  | number of results we want to fetch             | int           | 5               |
| start_time  | start time of the range to fetch all schedules | string        | 2023-06-28 10:00:00              |
| end_time  | end time of the range to fetch all schedules   | string        | 2023-07-02 12:00:00              |
| continuationToken  | token to fetch the next set of schedules       | string        | 19000474657374000004000000 |
| continuationStartTime  | startTime from where we continue fetching next set of schedules | long             | 1687930200               |
| status  | status type of the schedules we want to fetch  | string        | ERROR               |


**Note** : ContinuationToken and continuationStartTime are generated after the first call, it's not needed for the first time api call.

### Curl
```bash
curl --location --request GET 'http://localhost:8080/goscheduler/apps/revamp/schedules?size=5&start_time=2023-06-28 10:00:00&status=SUCCESS&end_time=2023-07-02 12:00:00' \
--header 'Accept: application/json' \
--header 'Authorization: Basic ZXJwYWRtaW46d2VsY29tZUAyNTg='
```

### Sample Success Response: 200

```json
{
    "status": {
        "statusCode": 200,
        "statusMessage": "Success",
        "statusType": "Success",
        "totalCount": 0
    },
    "data": {
        "schedules": null,
        "continuationToken": "",
        "continuationStartTime": 1648808400
    }
}
```

### Sample Error Response: 400
```json
{
    "status": {
        "statusCode": 400,
        "statusMessage": "Time range of more than 30 days is not allowed",
        "statusType": "Fail"
    }
}
```

## 10. Deactivate App API
This API handles the deactivation of an application by updating its active status to "false" in the database, deactivating the application in the supervisor, and returning an appropriate response indicating the deactivation status.
On deactivation, all the pollers will be stopped, so no new schedules could be created and no schedules will be triggered.
### Method: POST
```
http://localhost:8080/goscheduler/apps/revamp/deactivate
```

### Curl
```bash
curl --location --request POST 'http://localhost:8080/goscheduler/apps/revamp/deactivate' \
--header 'Accept: application/json' \
--header 'Authorization: Basic ZXJwYWRtaW46d2VsY29tZUAyNTg='
```

### Sample Success Response: 200
```json
{
    "status": {
        "statusCode": 201,
        "statusMessage": "Success",
        "statusType": "Success",
        "totalCount": 0
    },
    "data": {
        "appId": "revamp",
        "Active": false
    }
}
```
### Sample Error Resonse: 400
```json
{
    "status": {
        "statusCode": 4001,
        "statusMessage": "unregistered App",
        "statusType": "Fail"
    }
}
```

## 11. Activate App API
This API handles the activation of an application by updating its active status in the database, activating the application in the supervisor, and returning an appropriate response indicating the activation status.

It checks if the `Active` field of the application is already set to `true`. If it is `true`, it records the activation failure and returns an appropriate response indicating that the app is already activated.
### Method: POST
```
http://localhost:8080/goscheduler/apps/revamp/activate
```

### Curl
```bash
curl --location --request POST 'http://localhost:8080/goscheduler/apps/revamp/activate' \
--header 'Content-Type: application/x-www-form-urlencoded' \
--header 'Accept: application/json' \
--header 'Authorization: Basic ZXJwYWRtaW46d2VsY29tZUAyNTg='
```

### Sample Success Response: 200
```json
{
    "status": {
        "statusCode": 201,
        "statusMessage": "Success",
        "statusType": "Success",
        "totalCount": 0
    },
    "data": {
        "appId": "revamp",
        "Active": true
    }
}
```
### Sample Error Response: 4XX
```json
{
    "status": {
        "statusCode": 4001,
        "statusMessage": "unregistered App",
        "statusType": "Fail"
    }
}
```

```json
{
    "status": {
        "statusCode": 4003,
        "statusMessage": "app is already activated",
        "statusType": "Fail"
    }
}
```

## 12. Reconcile/Bulk Action API
This API gets all the schedules of an app in bulk based on time range and status.

This HTTP endpoint that handles bulk actions for schedules of an app based on status.

Based on `actionType`, it performs the following actions:

- If it is `reconcile`, it retriggers all the schedules of an app again. If it is `Delete`, it deletes all the schedules of the app in bulk.
- If the `actionType` is invalid, handle the error and return an appropriate response indicating the invalid action type.

It parses the request to extract the time range and status parameters. The end time of the time range should be before the start time and the duration of the time range should not exceed the maximum allowed period.
### Method: POST
```
http://localhost:8080/goscheduler/apps/revamp/bulk-action/reconcile?status=SUCCESS&start_time=2023-02-06%2010:47:00&end_time=2023-02-06%2011:50:00
```

### Query Params


| Param | Description                                   | Type of Value | Example             |
|-------|-----------------------------------------------|---------------|---------------------|
| status  | status type of the schedules we want to fetch | string        | SUCCESS             |
| start_time  | start time of the range to fetch all schedules | string        | 2023-02-06 10:47:00 |
| end_time  | end time of the range to fetch all schedules  | string        | 2023-02-06 11:50:00 |

### Curl
```bash
curl --location --request POST 'http://localhost:8080/goscheduler/apps/revamp/bulk-action/reconcile?status=SUCCESS&start_time=2023-02-06%2010:47:00&end_time=2023-02-06%2011:50:00' \
--header 'Accept: application/json' \
--header 'Authorization: Basic ZXJwYWRtaW46d2VsY29tZUAyNTg='
```

### Sample Success Response : 200

```json
{
    "status": {
        "statusCode": 200,
        "statusMessage": "Success",
        "statusType": "Success",
        "totalCount": 0
    },
    "remarks": "reconcile initiated successfully for app: revamp, timeRange: {StartTime:2023-02-06 10:47:00 +0530 IST EndTime:2023-02-06 11:50:00 +0530 IST}, status: SUCCESS"
}
```

## 13. Delete Schedule API
This API cancels a schedule based on its ID, handles different deletion scenarios, and returns the appropriate response with the deleted schedule or error information.
On deleting a cron schedule, all the children runs will also be deleted.

After a particular schedule is deleted, if we run this delete schedule API again, it would give "No Schedules found".
### Method: DELETE
```
http://localhost:8080/goscheduler/schedules/1497b35c-1a21-11ee-8689-ceaebc99522c
```

### Curl

```bash
curl --location --request DELETE 'http://localhost:8080/goscheduler/schedules/1497b35c-1a21-11ee-8689-ceaebc99522c' \
--header 'Authorization: Basic ZXJwYWRtaW46d2VsY29tZUAyNTg='
```

### Sample Success Response: 200
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
            "scheduleId": "1497b35c-1a21-11ee-8689-ceaebc99522c",
            "payload": "{}",
            "appId": "revamp",
            "scheduleTime": 2529772970,
            "Ttl": 0,
            "partitionId": 1,
            "scheduleGroup": 2529772920,
            "httpCallback": {
                "url": "http://127.0.0.1:8080/goscheduler/healthcheck",
                "method": "GET",
                "headers": {
                    "Accept": "application/json",
                    "Content-Type": "application/json"
                }
            }
        }
    }
}
```
___________________________________________________

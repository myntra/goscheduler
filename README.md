# GoScheduler
GoScheduler, also known as Myntra's Scheduler Service (MySS), is an open-source project designed to handle high throughput with low latency for scheduled job executions. GoScheduler is based on [Uber Ringpop](https://github.com/uber/ringpop-go) and offers capabilities such as multi-tenancy, per-minute granularity, horizontal scalability, fault tolerance, and other essential features. GoScheduler is written in Golang and utilizes Cassandra DB, allowing it to handle high levels of concurrent create/delete and callback throughputs. Further information about GoScheduler can be found in this [article](https://medium.com/myntra-engineering/myntra-scheduler-service-a0153a04526c).

# Architecture
![Go Scheduler Architecture](./docs/images/go_scheduler_arch.png)


# Design Overview
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

## Poller Cluster
The Poller Cluster in the Scheduler service utilizes the [Uber ringpop-go library](https://github.com/uber/ringpop-go) for its implementation. Ringpop provides application-level sharding, creating a consistent hash ring of available Poller Cluster nodes. The ring ensures that keys are distributed across the ring, with specific parts of the ring owned by individual Poller Cluster nodes.

### Poller Distribution
Every client within the Scheduler service owns a fixed number of Poller instances. Let's consider the total number of Poller instances assigned to all clients across all nodes as X. If there are Y clients where each client owns C1x, C2x, ..., CYx Poller instances respectively (where C1x + C2x + ... + CYx = X), and there are N Poller Cluster nodes, then each node would run approximately X/N Poller instances (i.e., X/N = C1x/N + C2x/N + ... + CYx/N).

### Scalability and Fault Tolerance
The Poller Cluster exhibits scalability and fault tolerance characteristics. When a node goes down, X/N Poller instances automatically shift to the available N-1 nodes, maintaining the distribution across the remaining nodes. Similarly, when a new node is added to the cluster, X/(N+1) Poller instances are shifted to the new node, while each existing node gives away X/N - X/(N+1) Poller instances.

This approach ensures load balancing and fault tolerance within the Poller Cluster, enabling efficient task execution and distribution across the available nodes.


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

## Usage

### Client onboarding

Use the following API to create an app:

```bash
curl --location 'http://localhost:8080/myss/app' \
--header 'Content-Type: application/json' \
--data '{
    "appId": "Athena",
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
        "appId": "Athena",
        "partitions": 5,
        "active": true,
        "configuration": {}
    }
}
```

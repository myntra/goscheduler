# GoScheduler
GoScheduler, also known as Myntra's Scheduler Service (MySS), is an open-source project designed to handle high throughput with low latency for scheduled job executions. GoScheduler is based on [Uber Ringpop](https://github.com/uber/ringpop-go) and offers capabilities such as multi-tenancy, per-minute granularity, horizontal scalability, fault tolerance, and other essential features. GoScheduler is written in Golang and utilizes Cassandra DB, allowing it to handle high levels of concurrent create/delete and callback throughputs. Further information about GoScheduler can be found in this [article](https://medium.com/myntra-engineering/myntra-scheduler-service-a0153a04526c).

# Architecture
![Go Scheduler Architecture](./docs/images/go_scheduler_arch.png)
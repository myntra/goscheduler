CREATE KEYSPACE IF NOT EXISTS schedule_management WITH replication = {'class': 'SimpleStrategy', 'replication_factor' : 3} AND durable_writes = true;

CREATE TABLE IF NOT EXISTS schedule_management.schedules (
                                              app_id text,
                                              partition_id int,
                                              schedule_time_group timestamp,
                                              schedule_id uuid,
                                              callback_type text,
                                              callback_details text,
                                              payload text,
                                              schedule_time timestamp,
                                              parent_schedule_id uuid,
                                              PRIMARY KEY ((app_id, partition_id, schedule_time_group), schedule_id)
) WITH CLUSTERING ORDER BY (schedule_id DESC);

CREATE MATERIALIZED VIEW IF NOT EXISTS schedule_management.view_schedules AS
SELECT schedule_id, app_id, partition_id, schedule_time_group, callback_type, callback_details, payload, schedule_time, parent_schedule_id
FROM schedule_management.schedules
WHERE schedule_id IS NOT NULL AND app_id IS NOT NULL AND partition_id IS NOT NULL AND schedule_time_group IS NOT NULL
PRIMARY KEY (schedule_id, app_id, partition_id, schedule_time_group)
WITH CLUSTERING ORDER BY (app_id ASC, partition_id ASC, schedule_time_group ASC);

CREATE TABLE IF NOT EXISTS schedule_management.status (
                                           app_id text,
                                           partition_id int,
                                           schedule_time_group timestamp,
                                           schedule_id uuid,
                                           schedule_status text,
                                           error_msg text,
                                           reconciliation_history text,
                                           PRIMARY KEY ((app_id, partition_id), schedule_id)
) WITH CLUSTERING ORDER BY (schedule_id DESC);

CREATE TABLE IF NOT EXISTS schedule_management.recurring_schedules_by_id (
                                                              app_id text,
                                                              partition_id int,
                                                              schedule_id uuid,
                                                              callback_type text,
                                                              callback_details text,
                                                              payload text,
                                                              cron_expression text,
                                                              status text,
                                                              PRIMARY KEY (schedule_id)
);

CREATE TABLE IF NOT EXISTS schedule_management.recurring_schedules_by_partition (
                                                                     app_id text,
                                                                     partition_id int,
                                                                     schedule_id uuid,
                                                                     callback_type text,
                                                                     callback_details text,
                                                                     payload text,
                                                                     cron_expression text,
                                                                     status text,
                                                                     PRIMARY KEY (partition_id, schedule_id, app_id)
);

CREATE TABLE IF NOT EXISTS schedule_management.recurring_schedule_runs (
                                                            app_id text,
                                                            partition_id int,
                                                            schedule_time_group timestamp,
                                                            schedule_id uuid,
                                                            callback_type text,
                                                            callback_details text,
                                                            payload text,
                                                            schedule_time timestamp,
                                                            parent_schedule_id uuid,
                                                            PRIMARY KEY (parent_schedule_id, schedule_time_group)
) WITH CLUSTERING ORDER BY (schedule_time_group DESC);

CREATE KEYSPACE IF NOT EXISTS cluster WITH replication = {'class': 'SimpleStrategy', 'replication_factor': '3'}  AND durable_writes = true;

CREATE TABLE IF NOT EXISTS cluster.entity (
                                              id text,
                                              nodename text,
                                              status int,
                                              history text,
                                              PRIMARY KEY (id)
);

CREATE TABLE IF NOT EXISTS cluster.apps (
                                            id text,
                                            partitions int,
                                            active boolean,
                                            PRIMARY KEY (id)
);

CREATE MATERIALIZED VIEW IF NOT EXISTS cluster.nodes AS
SELECT nodename, id, status
FROM cluster.entity
WHERE nodename IS NOT NULL AND id IS NOT NULL
PRIMARY KEY (nodename, id);
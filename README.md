# Rebound

Rebound is a reliable message queue powered by SQLite, designed for simplicity
and persistence.

### Key Features:

- Intuitive HTTP API
- Ensures job retention during server downtime
- Supports easy data backup through tools like
  [Litestream](https://litestream.io/)

## Installation

Install Rebound using Docker:

```sh
docker run -p 3000:3000 -v ./data:/data ...
```

## How to Use

Rebound is operated via an HTTP API. **Note**: Rebound does not have built-in
authentication, so avoid exposing it directly to the internet.

### Workflow Overview

A job in Rebound transitions through three main stages:

#### Adding a Job

Jobs are first added to a queue. Rebound supports job prioritization, where a
lower value indicates higher priority (e.g., priority `0` is higher than `1`).
Jobs can also be delayed for a specified time before being added to the queue,
using a format like `1m` or `50s` (see
[Go documentation](https://pkg.go.dev/time#ParseDuration) for duration formats).
Each job includes a Time To Run (TTR) parameter that determines how long a
worker has to process it. If a worker fails to complete the job within the TTR,
the job is returned to the queue for other workers to handle. Ensure that the
TTR is set appropriately to prevent a job from being processed multiple times.

To add a job, send a `POST` request to `localhost:3000/{queue-name}` with the
following JSON payload:

```jsonc
{
  "priority": 5,  // Optional, default: 0
  "delay": "1m",  // Optional, default: no delay
  "ttr": "5m",    // Optional, default: 2 minutes
  "body": "This is a job"
}
```

#### Reserving a Job

Workers reserve jobs to process them. Once reserved, the worker has the TTR
duration to complete the job. If the job is not finished in that time (e.g., the
worker crashes), it is placed back in the queue.

To reserve a job, send a `GET` request to `localhost:3000/{queue-name}`. The
response will be structured as:

```jsonc
{
  "id": 1,
  "queue": "{queue-name}",
  "priority": 5,
  "body": "This is a job"
}
```

If no job is currently available, a `204 No Content` response is returned.

#### Deleting a Job

When a job is successfully completed, the worker should delete it by sending a
`DELETE` request to `localhost:3000/{queue-name}/{job-id}`.

## Contributing

Contributions are welcome! Feel free to submit pull requests.

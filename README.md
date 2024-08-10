# Redis Scheduler

Redis Scheduler is a microservice designed to schedule and manage timed tasks using Redis. It allows you to schedule tasks that will execute after a specified TTL (Time-To-Live) by triggering a webhook. If the webhook fails, the task will be retried based on configurable parameters.

## Why Use Redis Scheduler?

### The Problems with Intervals and Cron Jobs

While intervals and cron jobs are commonly used for scheduling tasks, they come with several inherent issues:

1. **Lack of Scalability**: 
   - Cron jobs run on a single server, which can create bottlenecks if the load increases. In contrast, Redis Scheduler can scale horizontally across multiple instances, distributing the workload effectively.

2. **Single Point of Failure**: 
   - If the server running the cron job goes down, all scheduled tasks are lost or delayed. Redis Scheduler leverages Redis for persistence, allowing tasks to be retried even if the service restarts or crashes.

3. **Difficult to Manage Dependencies**:
   - When multiple tasks depend on each other, managing execution order and timing can become complex. Redis Scheduler handles task dependencies more gracefully by allowing you to schedule tasks dynamically based on their success or failure.

4. **Resource Overhead**: 
   - Cron jobs can consume unnecessary resources if they are running tasks at frequent intervals, even when there are no tasks to execute. Redis Scheduler can run tasks based on actual need, improving resource efficiency.

5. **Limited Monitoring and Feedback**: 
   - Cron jobs typically lack robust monitoring and error handling mechanisms. With Redis Scheduler, you can implement comprehensive logging and alerting based on the success or failure of each task, making it easier to diagnose issues.

6. **Complex Time Calculations**:
   - Handling time zones and daylight saving changes can introduce errors in cron schedules. Redis Scheduler uses a simple TTL mechanism, making it straightforward and reliable.

### Why Choose Redis Scheduler?

By using Redis Scheduler, you gain:

- **Reliability**: Persistent storage and automatic retries ensure tasks are not lost.
- **Scalability**: Easily handle increased workloads across multiple instances.
- **Simplicity**: A clean and easy-to-use API for scheduling tasks without the overhead of cron or interval-based systems.
- **Flexibility**: Easily modify and manage tasks based on application needs.

## Features

- Schedule tasks with a TTL and webhook URL.
- Automatically trigger a webhook when the TTL expires.
- Retry mechanism with configurable retry count and intervals.
- Easy deployment with Docker and Docker Compose.

## Requirements

- Docker installed on your server or local machine.
- Docker Compose (optional, but recommended).

## Environment Variables

Make sure to set the following environment variables before running the service:

- `REDIS_URL`: URL of the Redis server (e.g., `redis://localhost:6379`).
- `API_AUTH`: Authorization token that is required for API requests and webhook validation.
- `PORT`: Port on which the service will run (default is `8080`).
- `RETRIES`: Number of retry attempts for failed webhooks.
- `RETRY_TIME`: Time (in seconds) between retries.

## Running with Docker

You can easily run Redis Scheduler using Docker. Follow these steps:

<details>
<summary>1. Pull the Docker Image</summary>

```bash
docker pull ghcr.io/digital39999/redis-scheduler:latest
```

</details>

<details>
<summary>2. Run the Container</summary>

Run the container with the necessary environment variables:

```bash
docker run -d \
  -e REDIS_URL="redis://your-redis-url:6379" \
  -e API_AUTH="your-api-auth-token" \
  -e PORT=8080 \
  -e RETRIES=5 \
  -e RETRY_TIME=60 \
  -p 8080:8080 \
  ghcr.io/digital39999/redis-scheduler:latest
```

</details>

<details>
<summary>3. Access the Service</summary>

The service will be available at `http://localhost:8080`.

</details>

## Running with Docker Compose

If you prefer to use Docker Compose, follow these steps:

<details>
<summary>1. Create a `docker-compose.yml` File</summary>

Here’s an example `docker-compose.yml`:

```yaml
version: '3.8'

services:
  redis-scheduler:
    image: ghcr.io/digital39999/redis-scheduler:latest
    environment:
      REDIS_URL: "redis://redis:6379"
      API_AUTH: "your-api-auth-token"
      PORT: 8080
      RETRIES: 5
      RETRY_TIME: 60
    ports:
      - "8080:8080"
    depends_on:
      - redis

  redis:
    image: redis:latest
    ports:
      - "6379:6379"
```

</details>

<details>
<summary>2. Run the Services</summary>

To start the services, use the following command:

```bash
docker-compose up -d
```

</details>

<details>
<summary>3. Access the Service</summary>

Once the services are up, you can access Redis Scheduler at `http://localhost:8080`.

</details>

## API Usage

<details>
<summary>Routes Overview</summary>

### Schedule a Task

You can schedule a task by sending a POST request to `/schedule`. Here's an example using `curl`:

```bash
curl -X POST http://localhost:8080/schedule \
-H "Authorization: your-api-auth-token" \
-H "Content-Type: application/json" \
-d '{
  "webhook": "https://example.com/webhook",
  "ttl": 120,
  "data": {
    "message": "Hello, World!"
  }
}'
```

- **`webhook`**: The URL to trigger when the TTL expires.
- **`ttl`**: Time-to-live in seconds after which the webhook will be triggered.
- **`data`**: Any JSON data you want to send to the webhook.

### Get All Active Tasks

To retrieve a list of active tasks, send a GET request to `/schedules`:

```bash
curl -X GET http://localhost:8080/schedules \
-H "Authorization: your-api-auth-token"
```

### Get Task Details

To get details of a specific task, send a GET request to `/schedule/:key`:

```bash
curl -X GET http://localhost:8080/schedule/:key \
-H "Authorization: your-api-auth-token"
```

- Replace `:key` with the task key you want to retrieve.

### Update a Task

To update an existing scheduled task, send a PATCH request to `/schedule/:key`:

```bash
curl -X PATCH http://localhost:8080/schedule/:key \
-H "Authorization: your-api-auth-token" \
-H "Content-Type: application/json" \
-d '{
  "webhook": "https://example.com/new-webhook",
  "ttl": 180,
  "data": {
    "message": "Updated message"
  }
}'
```

- **Fields that can be updated**:
  - **`webhook`**: New URL to trigger.
  - **`ttl`**: Updated time-to-live in seconds.
  - **`data`**: New JSON data to send to the webhook.

### Cancel a Task

To cancel a scheduled task, send a DELETE request to `/schedule/:key`:

```bash
curl -X DELETE http://localhost:8080/schedule/:key \
-H "Authorization: your-api-auth-token"
```

### Get System Statistics

To retrieve system statistics, send a GET request to `/stats`:

```bash
curl -X GET http://localhost:8080/stats \
-H "Authorization: your-api-auth-token"
```

- This will return information such as the number of schedules running, total Redis keys, and microservices CPU and RAM usage.

### Purge All Schedules

To delete all scheduled tasks, send a POST request to `/purge`:

```bash
curl -X POST http://localhost:8080/purge \
-H "Authorization: your-api-auth-token"
```

</details>

<details>
<summary>Examples</summary>

### Example Node.js Client

Here’s how you could integrate Redis Scheduler into a Node.js project:

```javascript
const apiUrl = 'http://localhost:8080/schedule';
const apiToken = 'your_api_auth_token';

async function scheduleTask() {
  try {
    const response = await fetch(apiUrl, {
      method: 'POST',
      headers: {
        'Authorization': apiToken,
        'Content-Type': 'application/json'
      },
      body: JSON.stringify({
        webhook: 'https://example.com/webhook',
        ttl: 120, // 2 minutes
        data: {
          message: 'Hello from Node.js!'
        }
      })
    }).then(res => res.json());

    if (response.error) throw new Error(response.error);
    console.log('Task scheduled successfully:', data);
  } catch (error) {
    console.error('Error scheduling task:', error.message);
  }
}

scheduleTask();
```

### Example Python Client

Here’s how you could integrate Redis Scheduler into a Python project using `requests`:

```python
import requests

api_url = 'http://localhost:8080/schedule'
api_token = 'your_api_auth_token'

def schedule_task():
    headers = {
        'Authorization': api_token,
        'Content-Type': 'application/json'
    }
    data = {
        'webhook': 'https://example.com/webhook',
        'ttl': 120,  # 2 minutes
        'data

': {
            'message': 'Hello from Python!'
        }
    }
    
    response = requests.post(api_url, headers=headers, json=data)
    if response.status_code == 200:
        print('Task scheduled successfully:', response.json())
    else:
        print('Error scheduling task:', response.text)

schedule_task()
```

</details>

## Contributing

If you'd like to contribute to this project, feel free to open a pull request or submit an issue on the GitHub repository.

## License

This project is licensed under the MIT License. See the [LICENSE](LICENSE) file for more details.
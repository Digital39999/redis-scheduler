# Redis Scheduler

Redis Scheduler is a microservice designed to schedule and manage timed tasks using Redis. It allows you to schedule tasks that will execute after a specified TTL (Time-To-Live) by triggering a webhook. If the webhook fails, the task will be retried based on configurable parameters.

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

### 1. Pull the Docker Image

```bash
docker pull ghcr.io/digital39999/redis-scheduler:latest
```

### 2. Run the Container

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

### 3. Access the Service

The service will be available at `http://localhost:8080`.

## Running with Docker Compose

If you prefer to use Docker Compose, follow these steps:

### 1. Create a `docker-compose.yml` File

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

### 2. Run the Services

To start the services, use the following command:

```bash
docker-compose up -d
```

### 3. Access the Service

Once the services are up, you can access Redis Scheduler at `http://localhost:8080`.

## API Usage

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

### Example Node.js Client

Here’s how you could integrate Redis Scheduler into a Node.js project:

```javascript
const axios = require('axios');

const apiUrl = 'http://localhost:8080/schedule';
const apiToken = 'your_api_auth_token';

async function scheduleTask() {
  try {
    const response = await axios.post(apiUrl, {
      webhook: 'https://example.com/webhook',
      ttl: 120, // 2 minutes
      data: {
        message: 'Hello from Node.js!'
      }
    }, {
      headers: {
        'Authorization': apiToken,
        'Content-Type': 'application/json'
      }
    });

    console.log('Task scheduled successfully:', response.data);
  } catch (error) {
    console.error('Error scheduling task:', error.response ? error.response.data : error.message);
  }
}

scheduleTask();
```

### Using the Scheduled Task

Once the task is scheduled, Redis Scheduler will attempt to post the provided data to the specified webhook after the TTL expires. If the webhook fails, it will retry based on the configured `RETRIES` and `RETRY_TIME` environment variables.

## Contributing

If you'd like to contribute to this project, feel free to open a pull request or submit an issue on the GitHub repository.

## License

This project is licensed under the MIT License. See the [LICENSE](LICENSE) file for more details.

# Redis Scheduler

`redis-scheduler` is a powerful Node.js package that allows you to manage scheduled tasks using Redis. With this package, you can create, retrieve, update, and delete scheduled tasks, as well as listen for webhook events.

## Table of Contents

- [Installation](#installation)
- [Examples](#examples)
- [License](#license)

## Installation

You can install `redis-scheduler` via npm:

```bash
npm install redis-scheduler
```

## Usage

### Standalone Setup

You can create an instance of the `RedisScheduler` with an internal webhook server by specifying the port. 

```typescript
import RedisScheduler from 'redis-scheduler';

const scheduler = new RedisScheduler({
    enableWebServerOnPort: 3000, // Port for the webhook server
    authorization: 'your-authorization-token',
    instanceUrl: 'https://your-instance-url.com', // Your instance URL
});
```

### Setup with Express

You can also use `redis-scheduler` with an existing Express application.

```typescript
import express from 'express';
import RedisScheduler from 'redis-scheduler';

const app = express();
const scheduler = new RedisScheduler({
    authorization: 'your-authorization-token',
    instanceUrl: 'https://your-instance-url.com', // Your instance URL
});

// Middleware to parse JSON
app.use(express.json());

// Custom webhook endpoint
app.post('/custom-webhook', (req, res) => {
    try {
        scheduler.onWebhook(req.body, req.headers['authorization'] || '');
        res.json({ status: 'success' });
    } catch (error) {
        res.status(400).json({ status: 'error', message: error.message });
    }
});

// Start your Express server
app.listen(3000, () => {
    console.log('Express server is running on http://localhost:3000');
});
```

## Examples

### Scheduling a Task

```typescript
const key = await scheduler.schedule({
    webhook: 'https://example.com/webhook',
    ttl: 60,
    data: { message: 'Hello, World!' }
});

console.log('Scheduled key:', key);
```

### Retrieving a Scheduled Task

```typescript
const schedule = await scheduler.getSchedule(key);
console.log('Scheduled task:', schedule);
```

### Updating a Scheduled Task

```typescript
const success = await scheduler.updateSchedule(key, {
    webhook: 'https://example.com/updated-webhook',
    ttl: 120,
    data: { message: 'Updated message' }
});
console.log('Update successful:', success);
```

### Deleting a Scheduled Task

```typescript
const success = await scheduler.deleteSchedule(key);
console.log('Delete successful:', success);
```

### Getting Statistics

```typescript
const stats = await scheduler.getStats();
console.log('Scheduler stats:', stats);
```

### Listening for Webhook Events

You can listen for webhook events using the event emitter feature. Here's an example:

```typescript
// Listening for incoming webhook events
scheduler.on('data', (data) => {
    console.log('Webhook event received:', data);
});

// Example webhook payload
const webhookPayload = {
    message: 'This is a webhook event!',
};

// Simulating webhook call
scheduler.onWebhook(webhookPayload, 'authorization-token-from-request-headers');
```

### TypeScript Custom Types

If you are using TypeScript, you can define custom types for your webhook events. Here’s an example:

```typescript
interface CustomWebhookData {
    message: string;
}

scheduler.on<CustomWebhookData>('data', (data) => {
    console.log('Webhook message:', data.message);
});

// Simulating webhook call with typed data
const webhookPayload: CustomWebhookData = {
    message: 'This is a typed webhook event!',
};

scheduler.onWebhook(webhookPayload, 'authorization-token-from-request-headers');
```

## License

This project is licensed under the MIT License - see the [LICENSE](https://github.com/Digital39999/redis-scheduler/LICENSE) file for details.
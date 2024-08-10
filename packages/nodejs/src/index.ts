import { ConstructorOptions, RawStatsData, RequestData, ResponseType, ScheduleData, StatsData } from './types';
import axios, { AxiosError, AxiosResponse } from 'axios';
import { serve } from '@hono/node-server';
import { EventEmitter } from 'events';
import { Hono } from 'hono';

export * from './types';

export default class RedisScheduler extends EventEmitter {
	private webhookServer: Hono | null = null;
	private options: ConstructorOptions;
	private webhookEnabled: boolean;

	constructor ({
		enableWebServerOnPort,
		authorization,
		instanceUrl,
	}: ConstructorOptions) {
		super();

		this.options = {
			enableWebServerOnPort,
			authorization,
			instanceUrl,
		};

		this.webhookEnabled = enableWebServerOnPort !== undefined;
		this.checkInstanceUrl(instanceUrl);

		if (this.webhookEnabled) {
			this.webhookServer = new Hono();

			this.webhookServer.post('/webhook', (c) => {
				const body = c.req.json();
				this.onWebhook(body, c.req.header('Authorization') || '');

				return c.json({ status: 200, message: 'Webhook received successfully.' }, 200);
			});

			serve({
				port: enableWebServerOnPort,
				fetch: this.webhookServer.fetch,
			});
		}
	}

	private async checkInstanceUrl(url: string): Promise<void> {
		if (!url) throw new Error('Instance URL is required.');

		const instanceUrl = new URL(url);
		if (!instanceUrl.hostname) throw new Error('Invalid instance URL.');

		const response = await axios.get(url).then((res) => res.data).catch((err: AxiosError) => err.response?.data);
		if (response.status !== 200) throw new Error('Invalid instance URL.');
		else if ('error' in response) throw new Error(response.error);

		return;
	}

	private async parseAxiosRequest<T>(response: Promise<AxiosResponse<ResponseType<T>>>): Promise<T> {
		const data = await response.then((res) => res.data).catch((err: AxiosError<ResponseType<T>>) => err.response?.data);

		if (!data || data.status !== 200) throw new Error('Request failed.');
		else if ('error' in data) throw new Error(data.error);

		return data.data;
	}

	private getHeaders(): Record<string, string> {
		return {
			'Authorization': this.options.authorization,
			'Content-Type': 'application/json',
		};
	}

	public async schedule(data: Omit<RequestData, 'retry'>): Promise<string> {
		return (await this.parseAxiosRequest<{ key: string; }>(axios({
			method: 'POST',
			url: `${this.options.instanceUrl}/schedule`,
			headers: this.getHeaders(),
			data,
		}))).key;
	}

	public async getSchedule<T>(key: string): Promise<ScheduleData<T>> {
		return await this.parseAxiosRequest<ScheduleData<T>>(axios({
			method: 'GET',
			url: `${this.options.instanceUrl}/schedule/${key}`,
			headers: this.getHeaders(),
		}));
	}

	public async getAllSchedules<T>(): Promise<ScheduleData<T>[]> {
		return await this.parseAxiosRequest<ScheduleData<T>[]>(axios({
			method: 'GET',
			url: `${this.options.instanceUrl}/schedules`,
			headers: this.getHeaders(),
		}));
	}

	public async updateSchedule(key: string, data: RequestData): Promise<boolean> {
		return !!(await this.parseAxiosRequest<string>(axios({
			method: 'PATCH',
			url: `${this.options.instanceUrl}/schedule/${key}`,
			headers: this.getHeaders(),
			data,
		})));
	}

	public async deleteSchedule(key: string): Promise<boolean> {
		return !!(await this.parseAxiosRequest<string>(axios({
			method: 'DELETE',
			url: `${this.options.instanceUrl}/schedule/${key}`,
			headers: this.getHeaders(),
		})));
	}

	public async deleteAllSchedules(): Promise<boolean> {
		return !!(await this.parseAxiosRequest<string>(axios({
			method: 'DELETE',
			url: `${this.options.instanceUrl}/schedules`,
			headers: this.getHeaders(),
		})));
	}

	public async getStats(): Promise<StatsData> {
		const data = await this.parseAxiosRequest<RawStatsData>(axios({
			method: 'GET',
			url: `${this.options.instanceUrl}/stats`,
			headers: this.getHeaders(),
		}));

		return {
			totalKeys: data.total_redis_keys,
			runningSchedules: data.running_schedules,

			cpuUsage: data.cpu_usage,
			ramUsage: data.ram_usage,

			ramUsageBytes: data.ram_usage_bytes,

			systemUptime: data.system_uptime,
			goRoutimeCount: data.go_routines,
		};
	}

	public onWebhook(body: unknown, authHeader: string): void {
		if (authHeader !== this.options.authorization) return;
		this.emit('data', body);
	}

	// Events.
	public on<T>(event: 'data', listener: (data: T) => void) {
		return super.on(event, listener);
	}

	public once<T>(event: 'data', listener: (data: T) => void) {
		return super.once(event, listener);
	}

	public off<T>(event: 'data', listener: (data: T) => void) {
		return super.off(event, listener);
	}

	public removeAllListeners(event?: string) {
		return super.removeAllListeners(event);
	}

	public listeners(event: string) {
		return super.listeners(event);
	}

	public listenerCount(event: string) {
		return super.listenerCount(event);
	}
}

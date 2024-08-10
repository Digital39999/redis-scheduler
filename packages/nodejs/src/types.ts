import { HttpStatusCode } from 'axios';

export type ResponseType<T> = {
	status: HttpStatusCode.Ok;
	data: T;
} | {
	status: Omit<HttpStatusCode, HttpStatusCode.Ok>;
	error: string;
}

export type RequestData<T = unknown> = {
	webhook: string;
	retry?: number;
	ttl: number;
	data: T;
}

export type ConstructorOptions = {
	enableWebServerOnPort?: number;
	authorization: string;
	instanceUrl: string;
}

export type ScheduleData<T = unknown> = {
	info: {
		key: string;
		ttl: number;
		retry: number;
		webhook: string;
		expires: string;
	};
	data: T;
}

export type StatsData = {
	totalKeys: number;
	runningSchedules: number;

	cpuUsage: number;
	ramUsage: string;

	ramUsageBytes: number;

	systemUptime: string;
	goRoutimeCount: number;
}

export type RawStatsData = {
	total_redis_keys: number;
	running_schedules: number;

	cpu_usage: number;
	ram_usage: string;

	ram_usage_bytes: number;

	system_uptime: string;
	go_routines: number;
}

import type { ZodType } from 'zod/v4';
import { fetcher, type RequestOptions } from '$lib/utils/fetcher.js';

interface EnvelopedResponse {
	data: unknown;
}

function unwrap(raw: unknown): unknown {
	if (raw && typeof raw === 'object' && 'data' in raw) {
		return (raw as EnvelopedResponse).data;
	}
	return raw;
}

export async function apiGet<T>(path: string, schema: ZodType<T>, signal?: AbortSignal): Promise<T> {
	const raw = await fetcher<unknown>(path, { signal });
	return schema.parse(unwrap(raw));
}

export async function apiPost<T>(
	path: string,
	body: unknown,
	schema: ZodType<T>,
	options?: Pick<RequestOptions, 'signal'>,
): Promise<T> {
	const raw = await fetcher<unknown>(path, { method: 'POST', body, ...options });
	return schema.parse(unwrap(raw));
}

export async function apiPut<T>(
	path: string,
	body: unknown,
	schema: ZodType<T>,
	options?: Pick<RequestOptions, 'signal'>,
): Promise<T> {
	const raw = await fetcher<unknown>(path, { method: 'PUT', body, ...options });
	return schema.parse(unwrap(raw));
}

export async function apiDelete(path: string, signal?: AbortSignal): Promise<void> {
	await fetcher<void>(path, { method: 'DELETE', signal });
}

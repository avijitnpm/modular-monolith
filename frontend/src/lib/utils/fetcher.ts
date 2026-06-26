import { env } from '$lib/config/env.js';
import type { ApiError } from '$lib/utils/errors.js';

export interface RequestOptions {
	method?: 'GET' | 'POST' | 'PUT' | 'PATCH' | 'DELETE';
	body?: unknown;
	headers?: Record<string, string>;
	signal?: AbortSignal;
}

export async function fetcher<T>(path: string, options: RequestOptions = {}): Promise<T> {
	const { method = 'GET', body, headers = {}, signal } = options;

	const url = `${env.apiBaseUrl}${path}`;
	const init: RequestInit = {
		method,
		headers: {
			'Content-Type': 'application/json',
			...headers
		},
		credentials: 'include',
		signal
	};

	if (body !== undefined) {
		init.body = JSON.stringify(body);
	}

	const res = await fetch(url, init);

	if (!res.ok) {
		const error = await parseErrorResponse(res);

		if (res.status === 401) {
			handleUnauthorized();
		}

		throw error;
	}

	if (res.status === 204) return undefined as T;
	return res.json() as Promise<T>;
}

function handleUnauthorized(): void {
	import('$lib/stores/auth.svelte.js').then(({ auth }) => {
		auth.logout();
	});
	if (typeof window !== 'undefined' && !window.location.pathname.startsWith('/login')) {
		window.location.href = '/login';
	}
}

async function parseErrorResponse(res: Response): Promise<ApiError> {
	try {
		const data = await res.json();
		return {
			status: res.status,
			code: data.code ?? 'ERROR',
			message: data.error ?? data.message ?? res.statusText,
			details: data.validation_errors
		};
	} catch {
		return {
			status: res.status,
			code: 'ERROR',
			message: res.statusText
		};
	}
}

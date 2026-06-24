export interface ApiError {
	status: number;
	code: string;
	message: string;
	details?: Record<string, string[]>;
}

export function normalizeError(err: unknown): ApiError {
	if (isApiError(err)) return err;
	if (err instanceof Error) {
		return { status: 0, code: 'UNKNOWN', message: err.message };
	}
	return { status: 0, code: 'UNKNOWN', message: 'An unexpected error occurred' };
}

export function isApiError(err: unknown): err is ApiError {
	return (
		typeof err === 'object' &&
		err !== null &&
		'status' in err &&
		'code' in err &&
		'message' in err
	);
}

export function getFieldErrors(err: ApiError, field: string): string[] {
	return err.details?.[field] ?? [];
}

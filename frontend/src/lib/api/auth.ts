import { apiGet } from '$lib/api/client.js';
import { AuthMeResponseSchema, type AuthMeResponse } from '$lib/schemas/auth.js';
import { fetcher } from '$lib/utils/fetcher.js';

export const authApi = {
	getMe(): Promise<AuthMeResponse> {
		return apiGet('/auth/me', AuthMeResponseSchema);
	},
	logout(): Promise<void> {
		return fetcher('/auth/logout', { method: 'POST' });
	},
};

import { apiGet } from '$lib/api/client.js';
import { UserListSchema, type UserList } from '$lib/schemas/user.js';

export const usersApi = {
	list(params?: { page?: number; limit?: number }): Promise<UserList> {
		const query = new URLSearchParams();
		if (params?.page) query.set('page', String(params.page));
		if (params?.limit) query.set('limit', String(params.limit));
		const qs = query.toString();
		return apiGet(`/users${qs ? `?${qs}` : ''}`, UserListSchema);
	}
};

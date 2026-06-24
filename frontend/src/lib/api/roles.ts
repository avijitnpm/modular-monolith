import { apiGet } from '$lib/api/client.js';
import { RoleListSchema, type RoleList } from '$lib/schemas/role.js';

export const rolesApi = {
	list(): Promise<RoleList> {
		return apiGet('/roles', RoleListSchema);
	},
};

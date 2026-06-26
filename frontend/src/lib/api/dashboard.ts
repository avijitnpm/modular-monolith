import { apiGet } from '$lib/api/client.js';
import { DashboardResponseSchema, type DashboardResponse } from '$lib/schemas/dashboard.js';

export const dashboardApi = {
	get(): Promise<DashboardResponse> {
		return apiGet('/organizations/dashboard', DashboardResponseSchema);
	}
};

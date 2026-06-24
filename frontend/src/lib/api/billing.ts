import { apiGet } from '$lib/api/client.js';
import {
	EntitlementsResponseSchema,
	SubscriptionSchema,
	UsageMetricsSchema,
	type EntitlementItem,
	type Subscription,
	type UsageMetrics,
} from '$lib/schemas/billing.js';

export const billingApi = {
	getSubscription(): Promise<Subscription | null> {
		return apiGet('/billing/subscription', SubscriptionSchema.nullable());
	},
	getUsage(): Promise<UsageMetrics> {
		return apiGet('/billing/usage', UsageMetricsSchema);
	},
	getEntitlements(): Promise<EntitlementItem[]> {
		return apiGet('/billing/entitlements', EntitlementsResponseSchema).then(
			(r) => r.entitlements,
		);
	},
};

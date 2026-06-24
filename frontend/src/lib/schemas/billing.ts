import { z } from 'zod/v4';

export const SubscriptionSchema = z.object({
	plan: z.string(),
	status: z.string(),
	provider: z.string(),
	current_period_end: z.string().nullable(),
});

export const UsageMetricsSchema = z.object({
	users: z.number(),
	documents: z.number(),
	api_requests: z.number(),
	storage: z.number(),
});

export const EntitlementItemSchema = z.object({
	metric: z.string(),
	used: z.number(),
	limit: z.number(),
	remaining: z.number(),
	allowed: z.boolean(),
});

export const EntitlementsResponseSchema = z.object({
	entitlements: z.array(EntitlementItemSchema),
});

export type Subscription = z.infer<typeof SubscriptionSchema>;
export type UsageMetrics = z.infer<typeof UsageMetricsSchema>;
export type EntitlementItem = z.infer<typeof EntitlementItemSchema>;

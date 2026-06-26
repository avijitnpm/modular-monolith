import { z } from 'zod/v4';

export const DashboardOrgSchema = z.object({
	id: z.string(),
	name: z.string()
});

export const DashboardSubscriptionSchema = z.object({
	plan: z.string(),
	status: z.string(),
	current_period_end: z.string().nullable()
});

export const DashboardUsageSchema = z.object({
	users: z.number(),
	documents: z.number(),
	api_requests: z.number(),
	storage: z.number()
});

export const DashboardEntitlementSchema = z.object({
	metric: z.string(),
	used: z.number(),
	limit: z.number(),
	remaining: z.number(),
	allowed: z.boolean()
});

export const DashboardResponseSchema = z.object({
	organization: DashboardOrgSchema,
	subscription: DashboardSubscriptionSchema.nullable(),
	usage: DashboardUsageSchema,
	entitlements: z.array(DashboardEntitlementSchema)
});

export type DashboardResponse = z.infer<typeof DashboardResponseSchema>;
export type DashboardEntitlement = z.infer<typeof DashboardEntitlementSchema>;
export type DashboardUsage = z.infer<typeof DashboardUsageSchema>;

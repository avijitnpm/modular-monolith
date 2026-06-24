import { z } from 'zod/v4';

export const PermissionSchema = z.object({
	id: z.string(),
	name: z.string(),
});

export const RoleSchema = z.object({
	id: z.string(),
	name: z.string(),
	permissions: z.array(PermissionSchema),
});

// Backend returns array directly (not wrapped in object)
export const RoleListSchema = z.array(RoleSchema);

export type Permission = z.infer<typeof PermissionSchema>;
export type Role = z.infer<typeof RoleSchema>;
export type RoleList = z.infer<typeof RoleListSchema>;

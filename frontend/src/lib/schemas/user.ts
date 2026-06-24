import { z } from 'zod/v4';

export const UserListItemSchema = z.object({
	id: z.string(),
	email: z.string(),
	name: z.string(),
	role: z.string(),
	status: z.string(),
	createdAt: z.string(),
});

export const UserListSchema = z.object({
	users: z.array(UserListItemSchema),
	total: z.number(),
});

export type UserListItem = z.infer<typeof UserListItemSchema>;
export type UserList = z.infer<typeof UserListSchema>;

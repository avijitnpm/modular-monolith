import { z } from 'zod/v4';

export const UserSchema = z.object({
	subject: z.string(),
	identity_id: z.string().optional(),
	email: z.string().optional(),
	email_verified: z.boolean().optional(),
	preferred_username: z.string().optional(),
	name: z.string().optional(),
	given_name: z.string().optional(),
	family_name: z.string().optional(),
	locale: z.string().optional(),
	organization_id: z.string().optional(),
	roles: z.array(z.string()).optional(),
});

export const AuthMeResponseSchema = z.object({
	authenticated: z.boolean(),
	user: UserSchema,
});

export type User = z.infer<typeof UserSchema>;
export type AuthMeResponse = z.infer<typeof AuthMeResponseSchema>;

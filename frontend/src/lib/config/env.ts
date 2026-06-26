export const env = {
	apiBaseUrl: import.meta.env.VITE_API_BASE_URL ?? '/api/v1',
	authLoginUrl: import.meta.env.VITE_AUTH_LOGIN_URL ?? '/api/v1/auth/login'
} as const;

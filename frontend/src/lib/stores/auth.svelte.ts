import { authApi } from '$lib/api/auth.js';
import type { User } from '$lib/schemas/auth.js';
import { session } from '$lib/stores/session.svelte.js';

class AuthStore {
	user = $state<User | null>(null);
	loading = $state(true);
	initialized = $state(false);

	get isAuthenticated(): boolean {
		return this.user !== null;
	}

	get organizationId(): string | undefined {
		return this.user?.organization_id;
	}

	async loadSession(): Promise<void> {
		this.loading = true;
		try {
			const data = await authApi.getMe();
			if (data.authenticated) {
				this.user = data.user;
			} else {
				this.user = null;
			}
		} catch {
			this.user = null;
			session.clear();
		} finally {
			this.loading = false;
			this.initialized = true;
		}
	}

	logout() {
		this.user = null;
		session.clear();
	}

	setUser(user: User | null) {
		this.user = user;
	}
}

export const auth = new AuthStore();

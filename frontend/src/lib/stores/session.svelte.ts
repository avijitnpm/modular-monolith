class SessionStore {
	expiresAt = $state<string | null>(null);

	get isValid(): boolean {
		if (!this.expiresAt) return false;
		return new Date(this.expiresAt).getTime() > Date.now();
	}

	setExpiry(expiresAt: string | null) {
		this.expiresAt = expiresAt;
	}

	clear() {
		this.expiresAt = null;
	}
}

export const session = new SessionStore();

class ThemeStore {
	mode = $state<'light' | 'dark'>('light');

	toggle() {
		this.mode = this.mode === 'light' ? 'dark' : 'light';
	}
}

export const theme = new ThemeStore();

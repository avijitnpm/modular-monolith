<script lang="ts">
	import { onMount } from 'svelte';
	import { goto } from '$app/navigation';
	import { resolve } from '$app/paths';
	import { auth } from '$lib/stores/auth.svelte.js';
	import { authApi } from '$lib/api/auth.js';

	onMount(async () => {
		try {
			await authApi.logout();
		} catch {
			// Ignore logout API errors
		}
		auth.logout();
		goto(resolve('/login'));
	});
</script>

<div class="flex min-h-screen items-center justify-center">
	<p class="text-muted-foreground text-sm">Signing out…</p>
</div>

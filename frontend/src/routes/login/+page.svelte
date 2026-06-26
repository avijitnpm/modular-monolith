<script lang="ts">
	import { Button } from '$lib/components/ui/button/index.js';
	import { env } from '$lib/config/env.js';
	import { auth } from '$lib/stores/auth.svelte.js';
	import { goto } from '$app/navigation';
	import { resolve } from '$app/paths';
	import { LogIn } from '@lucide/svelte';

	let loading = $state(false);

	$effect(() => {
		if (auth.isAuthenticated) {
			goto(resolve('/dashboard'));
		}
	});

	function handleLogin() {
		loading = true;
		window.location.href = env.authLoginUrl;
	}
</script>

<div class="flex min-h-screen items-center justify-center">
	<div class="w-full max-w-sm space-y-6 text-center">
		<div
			class="bg-primary text-primary-foreground mx-auto flex size-12 items-center justify-center rounded-lg text-xl font-bold"
		>
			M
		</div>
		<div>
			<h1 class="text-2xl font-semibold tracking-tight">Welcome back</h1>
			<p class="text-muted-foreground mt-1 text-sm">Sign in to your account to continue</p>
		</div>
		<Button class="w-full" size="lg" onclick={handleLogin} disabled={loading}>
			{#if loading}
				Redirecting…
			{:else}
				<LogIn class="mr-2 size-4" />
				Sign in
			{/if}
		</Button>
	</div>
</div>

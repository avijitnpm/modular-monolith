<script lang="ts">
	import type { Snippet } from 'svelte';
	import { auth } from '$lib/stores/auth.svelte.js';
	import { goto } from '$app/navigation';
	import { resolve } from '$app/paths';
	import AuthLoading from '$lib/components/shared/AuthLoading.svelte';

	interface Props {
		children: Snippet;
	}

	const { children }: Props = $props();

	$effect(() => {
		if (!auth.loading && !auth.isAuthenticated) {
			goto(resolve('/login'));
		}
	});
</script>

{#if auth.loading}
	<AuthLoading />
{:else if auth.isAuthenticated}
	{@render children()}
{/if}

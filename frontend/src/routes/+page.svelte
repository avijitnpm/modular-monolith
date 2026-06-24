<script lang="ts">
	import { goto } from '$app/navigation';
	import { onMount } from 'svelte';

	onMount(() => {
		goto('/dashboard');
	});
</script>

<p class="text-muted-foreground p-6 text-sm">Redirecting…</p>

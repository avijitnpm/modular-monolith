<script lang="ts">
	import { goto } from '$app/navigation';
	import { resolve } from '$app/paths';
	import { onMount } from 'svelte';

	onMount(() => {
		goto(resolve('/dashboard'));
	});
</script>

<p class="text-muted-foreground p-6 text-sm">Redirecting…</p>

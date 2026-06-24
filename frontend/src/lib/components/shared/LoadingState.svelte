<script lang="ts">
	import { Skeleton } from '$lib/components/ui/skeleton/index.js';

	interface Props {
		lines?: number;
	}

	const { lines = 3 }: Props = $props();
</script>

<div class="space-y-3">
	{#each Array(lines) as _}
		<Skeleton class="h-4 w-full" />
	{/each}
</div>

<script lang="ts">
	import type { Snippet } from 'svelte';

	interface Props {
		title: string;
		description?: string;
		actions?: Snippet;
	}

	const { title, description, actions }: Props = $props();
</script>

<div class="flex flex-col items-center justify-center py-12 text-center">
	<h3 class="text-lg font-medium">{title}</h3>
	{#if description}
		<p class="text-muted-foreground mt-1 text-sm">{description}</p>
	{/if}
	{#if actions}
		<div class="mt-4">
			{@render actions()}
		</div>
	{/if}
</div>

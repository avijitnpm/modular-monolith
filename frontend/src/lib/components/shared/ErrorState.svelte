<script lang="ts">
	import { Button } from '$lib/components/ui/button/index.js';
	import { AlertCircle } from '@lucide/svelte';

	interface Props {
		message?: string;
		onRetry?: () => void;
	}

	const { message = 'Something went wrong', onRetry }: Props = $props();
</script>

<div class="flex flex-col items-center justify-center py-12 text-center">
	<AlertCircle class="text-destructive mb-3 size-10" />
	<p class="text-muted-foreground text-sm">{message}</p>
	{#if onRetry}
		<Button variant="outline" size="sm" class="mt-4" onclick={onRetry}>Try again</Button>
	{/if}
</div>

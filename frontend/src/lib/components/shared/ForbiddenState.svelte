<script lang="ts">
	import { Button } from '$lib/components/ui/button/index.js';
	import { ShieldOff } from '@lucide/svelte';

	interface Props {
		message?: string;
	}

	const { message = 'You do not have permission to access this resource' }: Props = $props();
</script>

<div class="flex flex-col items-center justify-center py-12 text-center">
	<ShieldOff class="text-muted-foreground mb-3 size-10" />
	<h3 class="text-lg font-medium">Access Denied</h3>
	<p class="text-muted-foreground mt-1 text-sm">{message}</p>
	<Button variant="outline" size="sm" class="mt-4" onclick={() => history.back()}>Go back</Button>
</div>

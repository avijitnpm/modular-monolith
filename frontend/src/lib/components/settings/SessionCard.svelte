<script lang="ts">
	import { Button } from '$lib/components/ui/button/index.js';
	import { session } from '$lib/stores/session.svelte.js';
	import { formatDateTime } from '$lib/utils/date.js';
	import { resolve } from '$app/paths';

	interface Props {
		authenticated: boolean;
	}

	const { authenticated }: Props = $props();
</script>

<div class="bg-card text-card-foreground rounded-lg border p-4">
	<h3 class="text-sm font-medium">Session</h3>
	<div class="mt-3 space-y-2 text-sm">
		<div>
			<p class="text-muted-foreground">Authenticated</p>
			<p class="font-medium">{authenticated ? 'Yes' : 'No'}</p>
		</div>
		<div>
			<p class="text-muted-foreground">Session Active</p>
			<p class="font-medium">{session.isValid ? 'Yes' : 'No'}</p>
		</div>
		<div>
			<p class="text-muted-foreground">Expires At</p>
			<p class="font-medium">{session.expiresAt ? formatDateTime(session.expiresAt) : 'Unknown'}</p>
		</div>
	</div>
	<div class="mt-4">
		<a href={resolve('/logout')}>
			<Button variant="outline" size="sm">Logout</Button>
		</a>
	</div>
</div>

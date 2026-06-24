<script lang="ts">
	import type { Role } from '$lib/schemas/role.js';

	interface Props {
		role: Role;
	}

	const { role }: Props = $props();

	let expanded = $state(false);
</script>

<div class="bg-card text-card-foreground rounded-lg border p-4">
	<div class="flex items-center justify-between">
		<h3 class="font-medium capitalize">{role.name}</h3>
		<span class="text-muted-foreground text-sm">{role.permissions.length} permissions</span>
	</div>
	<button
		class="text-muted-foreground mt-2 text-sm underline-offset-2 hover:underline"
		onclick={() => (expanded = !expanded)}
	>
		{expanded ? 'Hide' : 'Show'} permissions
	</button>
	{#if expanded}
		<ul class="mt-2 space-y-1">
			{#each role.permissions as perm}
				<li class="text-muted-foreground text-sm font-mono">{perm.name}</li>
			{/each}
		</ul>
	{/if}
</div>

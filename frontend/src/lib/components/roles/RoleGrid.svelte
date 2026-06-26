<script lang="ts">
	import type { Role } from '$lib/schemas/role.js';
	import RoleCard from './RoleCard.svelte';

	interface Props {
		roles: Role[];
	}

	const { roles }: Props = $props();
</script>

<div class="grid gap-4 md:grid-cols-2">
	{#each roles as role (role.id)}
		<RoleCard {role} />
	{/each}
</div>

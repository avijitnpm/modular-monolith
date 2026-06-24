<script lang="ts">
	import { onMount } from 'svelte';
	import { usersApi } from '$lib/api/users.js';
	import type { UserListItem } from '$lib/schemas/user.js';
	import { isApiError } from '$lib/utils/errors.js';
	import { Button } from '$lib/components/ui/button/index.js';
	import PageHeader from '$lib/components/shared/PageHeader.svelte';
	import LoadingState from '$lib/components/shared/LoadingState.svelte';
	import ErrorState from '$lib/components/shared/ErrorState.svelte';
	import ForbiddenState from '$lib/components/shared/ForbiddenState.svelte';
	import EmptyState from '$lib/components/shared/EmptyState.svelte';
	import UserTable from '$lib/components/users/UserTable.svelte';
	import UserCard from '$lib/components/users/UserCard.svelte';

	let users = $state<UserListItem[]>([]);
	let loading = $state(true);
	let error = $state<string | null>(null);
	let forbidden = $state(false);
	let comingSoon = $state(false);

	async function load() {
		loading = true;
		error = null;
		forbidden = false;
		try {
			const result = await usersApi.list();
			users = result.users;
		} catch (err) {
			if (isApiError(err) && err.status === 403) {
				forbidden = true;
			} else {
				error = isApiError(err) ? err.message : 'Failed to load users';
			}
		} finally {
			loading = false;
		}
	}

	onMount(load);
</script>

{#if loading}
	<LoadingState lines={5} />
{:else if forbidden}
	<ForbiddenState />
{:else if error}
	<ErrorState message={error} onRetry={load} />
{:else}
	<div class="space-y-6">
		<PageHeader title="Users" description="Manage organization members.">
			{#snippet actions()}
				<Button size="sm" onclick={() => (comingSoon = true)}>Invite User</Button>
			{/snippet}
		</PageHeader>
		{#if comingSoon}
			<p class="text-muted-foreground text-sm">Coming Soon</p>
		{/if}
		{#if users.length === 0}
			<EmptyState title="No users found" description="Users will appear here once added." />
		{:else}
			<UserTable {users} />
			<div class="space-y-3 md:hidden">
				{#each users as user}
					<UserCard {user} />
				{/each}
			</div>
		{/if}
	</div>
{/if}

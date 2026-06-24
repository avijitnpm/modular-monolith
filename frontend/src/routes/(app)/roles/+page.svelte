<script lang="ts">
	import { onMount } from 'svelte';
	import { rolesApi } from '$lib/api/roles.js';
	import type { Role } from '$lib/schemas/role.js';
	import { isApiError } from '$lib/utils/errors.js';
	import PageHeader from '$lib/components/shared/PageHeader.svelte';
	import LoadingState from '$lib/components/shared/LoadingState.svelte';
	import ErrorState from '$lib/components/shared/ErrorState.svelte';
	import ForbiddenState from '$lib/components/shared/ForbiddenState.svelte';
	import EmptyState from '$lib/components/shared/EmptyState.svelte';
	import RoleGrid from '$lib/components/roles/RoleGrid.svelte';

	let roles = $state<Role[]>([]);
	let loading = $state(true);
	let error = $state<string | null>(null);
	let forbidden = $state(false);

	async function load() {
		loading = true;
		error = null;
		forbidden = false;
		try {
			roles = await rolesApi.list();
		} catch (err) {
			if (isApiError(err) && err.status === 403) {
				forbidden = true;
			} else {
				error = isApiError(err) ? err.message : 'Failed to load roles';
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
{:else if roles.length === 0}
	<EmptyState title="No roles found" description="Roles will appear here once configured." />
{:else}
	<div class="space-y-6">
		<PageHeader title="Roles" description="Roles and permissions available in this organization." />
		<RoleGrid {roles} />
	</div>
{/if}

<script lang="ts">
	import { onMount } from 'svelte';
	import { dashboardApi } from '$lib/api/dashboard.js';
	import type { DashboardResponse } from '$lib/schemas/dashboard.js';
	import { isApiError } from '$lib/utils/errors.js';
	import LoadingState from '$lib/components/shared/LoadingState.svelte';
	import ErrorState from '$lib/components/shared/ErrorState.svelte';
	import ForbiddenState from '$lib/components/shared/ForbiddenState.svelte';
	import DashboardHeader from '$lib/components/dashboard/DashboardHeader.svelte';
	import StatsGrid from '$lib/components/dashboard/StatsGrid.svelte';
	import SubscriptionCard from '$lib/components/dashboard/SubscriptionCard.svelte';
	import EntitlementsTable from '$lib/components/dashboard/EntitlementsTable.svelte';

	let data = $state<DashboardResponse | null>(null);
	let loading = $state(true);
	let error = $state<string | null>(null);
	let forbidden = $state(false);

	async function load() {
		loading = true;
		error = null;
		forbidden = false;
		try {
			data = await dashboardApi.get();
		} catch (err) {
			if (isApiError(err) && err.status === 403) {
				forbidden = true;
			} else {
				error = isApiError(err) ? err.message : 'Failed to load dashboard';
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
{:else if data}
	<div class="space-y-6">
		<DashboardHeader orgName={data.organization.name} />
		<StatsGrid usage={data.usage} />
		{#if data.subscription}
			<SubscriptionCard subscription={data.subscription} />
		{/if}
		{#if data.entitlements.length > 0}
			<EntitlementsTable entitlements={data.entitlements} />
		{/if}
	</div>
{/if}

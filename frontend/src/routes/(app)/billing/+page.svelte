<script lang="ts">
	import { onMount } from 'svelte';
	import { billingApi } from '$lib/api/billing.js';
	import type { Subscription, UsageMetrics, EntitlementItem } from '$lib/schemas/billing.js';
	import { isApiError } from '$lib/utils/errors.js';
	import PageHeader from '$lib/components/shared/PageHeader.svelte';
	import LoadingState from '$lib/components/shared/LoadingState.svelte';
	import ErrorState from '$lib/components/shared/ErrorState.svelte';
	import ForbiddenState from '$lib/components/shared/ForbiddenState.svelte';
	import EmptyState from '$lib/components/shared/EmptyState.svelte';
	import SubscriptionCard from '$lib/components/billing/SubscriptionCard.svelte';
	import UsageGrid from '$lib/components/billing/UsageGrid.svelte';
	import EntitlementsTable from '$lib/components/billing/EntitlementsTable.svelte';

	let subscription = $state<Subscription | null>(null);
	let usage = $state<UsageMetrics | null>(null);
	let entitlements = $state<EntitlementItem[]>([]);
	let loading = $state(true);
	let error = $state<string | null>(null);
	let forbidden = $state(false);

	async function load() {
		loading = true;
		error = null;
		forbidden = false;
		try {
			[subscription, usage, entitlements] = await Promise.all([
				billingApi.getSubscription(),
				billingApi.getUsage(),
				billingApi.getEntitlements()
			]);
		} catch (err) {
			if (isApiError(err) && err.status === 403) {
				forbidden = true;
			} else {
				error = isApiError(err) ? err.message : 'Failed to load billing';
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
{:else if !subscription}
	<EmptyState
		title="No subscription"
		description="No active subscription found for this organization."
	/>
{:else}
	<div class="space-y-6">
		<PageHeader title="Billing" description="Subscription, usage, and plan limits." />
		<SubscriptionCard {subscription} />
		{#if usage}
			<UsageGrid {usage} />
		{/if}
		{#if entitlements.length > 0}
			<EntitlementsTable {entitlements} />
		{/if}
	</div>
{/if}

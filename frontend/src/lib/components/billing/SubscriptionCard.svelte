<script lang="ts">
	import type { Subscription } from '$lib/schemas/billing.js';
	import { formatDate } from '$lib/utils/date.js';

	interface Props {
		subscription: Subscription;
	}

	const { subscription }: Props = $props();

	const statusColors: Record<string, string> = {
		active: 'bg-green-100 text-green-800',
		trialing: 'bg-blue-100 text-blue-800',
		cancelled: 'bg-red-100 text-red-800',
		past_due: 'bg-amber-100 text-amber-800',
	};

	const badgeClass = $derived(statusColors[subscription.status] ?? 'bg-gray-100 text-gray-800');
</script>

<div class="bg-card text-card-foreground rounded-lg border p-4">
	<h3 class="text-sm font-medium">Subscription</h3>
	<div class="mt-3 grid gap-4 text-sm sm:grid-cols-2 lg:grid-cols-4">
		<div>
			<p class="text-muted-foreground">Plan</p>
			<p class="font-medium capitalize">{subscription.plan}</p>
		</div>
		<div>
			<p class="text-muted-foreground">Status</p>
			<span class="mt-0.5 inline-block rounded-full px-2 py-0.5 text-xs font-medium {badgeClass}">
				{subscription.status.replace(/_/g, ' ')}
			</span>
		</div>
		<div>
			<p class="text-muted-foreground">Provider</p>
			<p class="font-medium capitalize">{subscription.provider}</p>
		</div>
		<div>
			<p class="text-muted-foreground">Current Period Ends</p>
			<p class="font-medium">
				{subscription.current_period_end ? formatDate(subscription.current_period_end) : '—'}
			</p>
		</div>
	</div>
</div>

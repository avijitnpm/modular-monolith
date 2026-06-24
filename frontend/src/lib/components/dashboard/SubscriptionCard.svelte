<script lang="ts">
	import type { DashboardResponse } from '$lib/schemas/dashboard.js';
	import { formatDate } from '$lib/utils/date.js';

	interface Props {
		subscription: NonNullable<DashboardResponse['subscription']>;
	}

	const { subscription }: Props = $props();
</script>

<div class="bg-card text-card-foreground rounded-lg border p-4">
	<h3 class="text-sm font-medium">Subscription</h3>
	<div class="mt-3 grid gap-2 text-sm sm:grid-cols-3">
		<div>
			<p class="text-muted-foreground">Plan</p>
			<p class="font-medium capitalize">{subscription.plan}</p>
		</div>
		<div>
			<p class="text-muted-foreground">Status</p>
			<p class="font-medium capitalize">{subscription.status}</p>
		</div>
		<div>
			<p class="text-muted-foreground">Current Period Ends</p>
			<p class="font-medium">
				{subscription.current_period_end ? formatDate(subscription.current_period_end) : '—'}
			</p>
		</div>
	</div>
</div>

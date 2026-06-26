<script lang="ts">
	import type { DashboardEntitlement } from '$lib/schemas/dashboard.js';

	interface Props {
		entitlements: DashboardEntitlement[];
	}

	const { entitlements }: Props = $props();

	function formatLimit(limit: number): string {
		return limit < 0 ? 'Unlimited' : limit.toLocaleString();
	}

	function formatRemaining(remaining: number, limit: number): string {
		if (limit < 0) return 'Unlimited';
		return remaining.toLocaleString();
	}
</script>

<div class="bg-card text-card-foreground rounded-lg border">
	<div class="p-4">
		<h3 class="text-sm font-medium">Entitlements</h3>
	</div>
	<div class="overflow-x-auto">
		<table class="w-full text-sm">
			<thead>
				<tr class="border-t text-left">
					<th class="px-4 py-2 font-medium">Metric</th>
					<th class="px-4 py-2 font-medium">Used</th>
					<th class="px-4 py-2 font-medium">Limit</th>
					<th class="px-4 py-2 font-medium">Remaining</th>
				</tr>
			</thead>
			<tbody>
				{#each entitlements as ent (ent.metric)}
					<tr class="border-t">
						<td class="px-4 py-2 capitalize">{ent.metric.replace(/_/g, ' ')}</td>
						<td class="px-4 py-2">{ent.used.toLocaleString()}</td>
						<td class="px-4 py-2">{formatLimit(ent.limit)}</td>
						<td class="px-4 py-2">{formatRemaining(ent.remaining, ent.limit)}</td>
					</tr>
				{/each}
			</tbody>
		</table>
	</div>
</div>

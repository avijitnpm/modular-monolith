<script lang="ts">
	import type { EntitlementItem } from '$lib/schemas/billing.js';

	interface Props {
		entitlements: EntitlementItem[];
	}

	const { entitlements }: Props = $props();

	function formatLimit(limit: number): string {
		return limit < 0 ? 'Unlimited' : limit.toLocaleString();
	}

	function formatRemaining(remaining: number, limit: number): string {
		if (limit < 0 || remaining < 0) return 'Unlimited';
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
					<th class="px-4 py-2 font-medium">Status</th>
				</tr>
			</thead>
			<tbody>
				{#each entitlements as ent (ent.metric)}
					<tr class="border-t">
						<td class="px-4 py-2 capitalize">{ent.metric.replace(/_/g, ' ')}</td>
						<td class="px-4 py-2">{ent.used.toLocaleString()}</td>
						<td class="px-4 py-2">{formatLimit(ent.limit)}</td>
						<td class="px-4 py-2">{formatRemaining(ent.remaining, ent.limit)}</td>
						<td class="px-4 py-2">
							{#if ent.allowed}
								<span
									class="inline-block rounded-full bg-green-100 px-2 py-0.5 text-xs font-medium text-green-800"
									>Available</span
								>
							{:else}
								<span
									class="inline-block rounded-full bg-red-100 px-2 py-0.5 text-xs font-medium text-red-800"
									>Limit Reached</span
								>
							{/if}
						</td>
					</tr>
				{/each}
			</tbody>
		</table>
	</div>
</div>

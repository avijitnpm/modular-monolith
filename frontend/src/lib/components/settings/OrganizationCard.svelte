<script lang="ts">
	interface Props {
		organizationId?: string;
	}

	const { organizationId }: Props = $props();
</script>

<div class="bg-card text-card-foreground rounded-lg border p-4">
	<h3 class="text-sm font-medium">Organization</h3>
	<div class="mt-3 space-y-2 text-sm">
		<div>
			<p class="text-muted-foreground">Organization ID</p>
			<p class="font-medium font-mono text-xs">{organizationId ?? '—'}</p>
		</div>
		<div>
			<p class="text-muted-foreground">Plan</p>
			<p class="text-muted-foreground italic">Not available</p>
		</div>
		<div>
			<p class="text-muted-foreground">Subscription Status</p>
			<p class="text-muted-foreground italic">Not available</p>
		</div>
	</div>
</div>

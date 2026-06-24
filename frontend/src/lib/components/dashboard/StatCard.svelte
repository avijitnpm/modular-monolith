<script lang="ts">
	interface Props {
		label: string;
		value: string;
	}

	const { label, value }: Props = $props();
</script>

<div class="bg-card text-card-foreground rounded-lg border p-4">
	<p class="text-muted-foreground text-sm">{label}</p>
	<p class="mt-1 text-2xl font-semibold">{value}</p>
</div>

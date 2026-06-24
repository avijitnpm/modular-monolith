<script lang="ts">
	interface Props {
		orgName: string;
	}

	const { orgName }: Props = $props();
</script>

<div>
	<h1 class="text-2xl font-semibold tracking-tight">Dashboard</h1>
	<p class="text-muted-foreground text-sm">{orgName}</p>
</div>

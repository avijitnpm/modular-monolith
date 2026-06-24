<script lang="ts">
	import StatCard from './StatCard.svelte';
	import type { DashboardUsage } from '$lib/schemas/dashboard.js';

	interface Props {
		usage: DashboardUsage;
	}

	const { usage }: Props = $props();

	function formatStorage(bytes: number): string {
		if (bytes < 1024) return `${bytes} B`;
		if (bytes < 1024 * 1024) return `${(bytes / 1024).toFixed(1)} KB`;
		if (bytes < 1024 * 1024 * 1024) return `${(bytes / (1024 * 1024)).toFixed(1)} MB`;
		return `${(bytes / (1024 * 1024 * 1024)).toFixed(1)} GB`;
	}
</script>

<div class="grid gap-4 sm:grid-cols-2 lg:grid-cols-4">
	<StatCard label="Users" value={usage.users.toLocaleString()} />
	<StatCard label="Documents" value={usage.documents.toLocaleString()} />
	<StatCard label="API Requests" value={usage.api_requests.toLocaleString()} />
	<StatCard label="Storage" value={formatStorage(usage.storage)} />
</div>

<script lang="ts">
	import type { UserListItem } from '$lib/schemas/user.js';
	import { formatDate } from '$lib/utils/date.js';

	interface Props {
		user: UserListItem;
	}

	const { user }: Props = $props();

	const statusColors: Record<string, string> = {
		active: 'bg-green-100 text-green-800',
		inactive: 'bg-gray-100 text-gray-800'
	};

	const roleColors: Record<string, string> = {
		owner: 'bg-purple-100 text-purple-800',
		admin: 'bg-blue-100 text-blue-800',
		member: 'bg-gray-100 text-gray-800',
		viewer: 'bg-amber-100 text-amber-800'
	};
</script>

<div class="bg-card text-card-foreground rounded-lg border p-4 md:hidden">
	<div class="flex items-center justify-between">
		<p class="font-medium">{user.name}</p>
		<span
			class="inline-block rounded-full px-2 py-0.5 text-xs font-medium {statusColors[user.status] ??
				'bg-gray-100 text-gray-800'}"
		>
			{user.status}
		</span>
	</div>
	<p class="text-muted-foreground mt-1 text-sm">{user.email}</p>
	<div class="mt-2 flex items-center gap-2 text-sm">
		<span
			class="inline-block rounded-full px-2 py-0.5 text-xs font-medium {roleColors[user.role] ??
				'bg-gray-100 text-gray-800'}"
		>
			{user.role}
		</span>
		<span class="text-muted-foreground">·</span>
		<span class="text-muted-foreground">{formatDate(user.createdAt)}</span>
	</div>
</div>

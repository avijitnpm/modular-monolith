<script lang="ts">
	import type { UserListItem } from '$lib/schemas/user.js';
	import { formatDate } from '$lib/utils/date.js';

	interface Props {
		users: UserListItem[];
	}

	const { users }: Props = $props();

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

<div class="bg-card text-card-foreground hidden rounded-lg border md:block">
	<div class="overflow-x-auto">
		<table class="w-full text-sm">
			<thead>
				<tr class="border-b text-left">
					<th class="px-4 py-3 font-medium">Name</th>
					<th class="px-4 py-3 font-medium">Email</th>
					<th class="px-4 py-3 font-medium">Role</th>
					<th class="px-4 py-3 font-medium">Status</th>
					<th class="px-4 py-3 font-medium">Created</th>
				</tr>
			</thead>
			<tbody>
				{#each users as user (user.id)}
					<tr class="border-b last:border-0">
						<td class="px-4 py-3 font-medium">{user.name}</td>
						<td class="px-4 py-3">{user.email}</td>
						<td class="px-4 py-3">
							<span
								class="inline-block rounded-full px-2 py-0.5 text-xs font-medium {roleColors[
									user.role
								] ?? 'bg-gray-100 text-gray-800'}"
							>
								{user.role}
							</span>
						</td>
						<td class="px-4 py-3">
							<span
								class="inline-block rounded-full px-2 py-0.5 text-xs font-medium {statusColors[
									user.status
								] ?? 'bg-gray-100 text-gray-800'}"
							>
								{user.status}
							</span>
						</td>
						<td class="px-4 py-3">{formatDate(user.createdAt)}</td>
					</tr>
				{/each}
			</tbody>
		</table>
	</div>
</div>

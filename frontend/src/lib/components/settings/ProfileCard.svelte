<script lang="ts">
	import type { User } from '$lib/schemas/auth.js';

	interface Props {
		user: User;
	}

	const { user }: Props = $props();
</script>

<div class="bg-card text-card-foreground rounded-lg border p-4">
	<h3 class="text-sm font-medium">Profile</h3>
	<div class="mt-3 space-y-2 text-sm">
		<div>
			<p class="text-muted-foreground">Name</p>
			<p class="font-medium">{user.name ?? '—'}</p>
		</div>
		<div>
			<p class="text-muted-foreground">Email</p>
			<p class="font-medium">{user.email ?? '—'}</p>
		</div>
		<div>
			<p class="text-muted-foreground">Organization ID</p>
			<p class="font-medium font-mono text-xs">{user.organization_id ?? '—'}</p>
		</div>
		<div>
			<p class="text-muted-foreground">User ID</p>
			<p class="font-medium font-mono text-xs">{user.subject}</p>
		</div>
	</div>
</div>

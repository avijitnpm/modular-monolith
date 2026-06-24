<script lang="ts">
	import { auth } from '$lib/stores/auth.svelte.js';
	import PageHeader from '$lib/components/shared/PageHeader.svelte';
	import LoadingState from '$lib/components/shared/LoadingState.svelte';
	import ErrorState from '$lib/components/shared/ErrorState.svelte';
	import ProfileCard from '$lib/components/settings/ProfileCard.svelte';
	import SessionCard from '$lib/components/settings/SessionCard.svelte';
	import AppearanceCard from '$lib/components/settings/AppearanceCard.svelte';
	import OrganizationCard from '$lib/components/settings/OrganizationCard.svelte';
</script>

{#if auth.loading}
	<LoadingState lines={5} />
{:else if !auth.user}
	<ErrorState message="Unable to load profile information." />
{:else}
	<div class="space-y-6">
		<PageHeader title="Settings" description="Manage your account preferences and session information." />
		<div class="grid gap-6 md:grid-cols-2">
			<div class="space-y-6">
				<ProfileCard user={auth.user} />
				<OrganizationCard organizationId={auth.organizationId} />
			</div>
			<div class="space-y-6">
				<SessionCard authenticated={auth.isAuthenticated} />
				<AppearanceCard />
			</div>
		</div>
	</div>
{/if}

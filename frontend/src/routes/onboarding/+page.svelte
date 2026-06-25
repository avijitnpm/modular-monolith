<script lang="ts">
	import { Button } from '$lib/components/ui/button/index.js';
	import { Input } from '$lib/components/ui/input/index.js';
	import { fetcher } from '$lib/utils/fetcher.js';
	import { isApiError } from '$lib/utils/errors.js';

	let orgName = $state('');
	let loading = $state(false);
	let error = $state<string | null>(null);

	async function handleSubmit(e: Event) {
		e.preventDefault();
		if (!orgName.trim()) return;

		loading = true;
		error = null;

		try {
			await fetcher('/onboarding', {
				method: 'POST',
				body: { organization_name: orgName.trim() },
			});
			window.location.href = '/dashboard';
		} catch (err) {
			if (isApiError(err)) {
				error = err.message;
			} else {
				error = 'Something went wrong. Please try again.';
			}
		} finally {
			loading = false;
		}
	}
</script>

<div class="flex min-h-screen items-center justify-center">
	<div class="w-full max-w-sm space-y-6 text-center">
		<div class="bg-primary text-primary-foreground mx-auto flex size-12 items-center justify-center rounded-lg text-xl font-bold">
			M
		</div>
		<div>
			<h1 class="text-2xl font-semibold tracking-tight">Create your organization</h1>
			<p class="text-muted-foreground mt-1 text-sm">Set up your workspace to get started</p>
		</div>
		<form onsubmit={handleSubmit} class="space-y-4">
			<Input
				type="text"
				placeholder="Organization name"
				bind:value={orgName}
				disabled={loading}
			/>
			{#if error}
				<p class="text-destructive text-sm">{error}</p>
			{/if}
			<Button class="w-full" size="lg" type="submit" disabled={loading || !orgName.trim()}>
				{#if loading}
					Creating…
				{:else}
					Create Organization
				{/if}
			</Button>
		</form>
	</div>
</div>

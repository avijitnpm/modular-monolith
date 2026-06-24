<script lang="ts">
	import { onMount } from 'svelte';
	import { SidebarProvider, SidebarInset } from '$lib/components/ui/sidebar/index.js';
	import AppSidebar from '$lib/components/layout/AppSidebar.svelte';
	import AppTopbar from '$lib/components/layout/AppTopbar.svelte';
	import AuthGuard from '$lib/components/shared/AuthGuard.svelte';
	import { auth } from '$lib/stores/auth.svelte.js';

	const { children } = $props();

	onMount(() => {
		auth.loadSession();
	});
</script>

<AuthGuard>
	<SidebarProvider>
		<AppSidebar />
		<SidebarInset>
			<AppTopbar />
			<main class="flex-1 p-6">
				{@render children()}
			</main>
		</SidebarInset>
	</SidebarProvider>
</AuthGuard>

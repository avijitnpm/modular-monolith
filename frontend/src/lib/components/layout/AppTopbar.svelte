<script lang="ts">
	import { SidebarTrigger } from '$lib/components/ui/sidebar/index.js';
	import { Separator } from '$lib/components/ui/separator/index.js';
	import { Avatar, AvatarFallback } from '$lib/components/ui/avatar/index.js';
	import { auth } from '$lib/stores/auth.svelte.js';
</script>

<header class="flex h-14 shrink-0 items-center gap-2 border-b px-4">
	<SidebarTrigger class="-ml-1" />
	<Separator orientation="vertical" class="mr-2 !h-4" />
	<div class="flex-1"></div>
	<Avatar class="size-8">
		<AvatarFallback>{auth.user?.name?.charAt(0) ?? '?'}</AvatarFallback>
	</Avatar>
</header>

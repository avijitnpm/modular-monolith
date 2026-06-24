<script lang="ts">
	import { Skeleton } from '$lib/components/ui/skeleton/index.js';
</script>

<div class="flex min-h-screen items-center justify-center">
	<div class="w-full max-w-xs space-y-4 text-center">
		<Skeleton class="mx-auto h-8 w-32" />
		<Skeleton class="mx-auto h-4 w-48" />
	</div>
</div>

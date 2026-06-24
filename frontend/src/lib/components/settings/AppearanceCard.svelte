<script lang="ts">
	import { Button } from '$lib/components/ui/button/index.js';
	import { theme } from '$lib/stores/theme.svelte.js';
</script>

<div class="bg-card text-card-foreground rounded-lg border p-4">
	<h3 class="text-sm font-medium">Appearance</h3>
	<div class="mt-3 flex items-center justify-between text-sm">
		<p class="text-muted-foreground">Theme</p>
		<Button variant="outline" size="sm" onclick={() => theme.toggle()}>
			{theme.mode === 'light' ? '☀️ Light' : '🌙 Dark'}
		</Button>
	</div>
</div>

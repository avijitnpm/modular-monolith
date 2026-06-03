<main>
	<section>
		<p class="eyebrow">Modular Monolith</p>
		<h1>Sign in</h1>
		<p class="summary">Use your Zitadel account to continue.</p>

		<a class="button" href="/api/v1/auth/login">Login with Zitadel</a>
	</section>
</main>

<style>
	:global(body) {
		margin: 0;
		font-family:
			Inter, ui-sans-serif, system-ui, -apple-system, BlinkMacSystemFont, "Segoe UI",
			sans-serif;
		background: #f7f7f4;
		color: #151515;
	}

	main {
		min-height: 100vh;
		display: grid;
		place-items: center;
		padding: 24px;
		box-sizing: border-box;
	}

	section {
		width: min(100%, 420px);
	}

	.eyebrow {
		margin: 0 0 12px;
		color: #5f6f52;
		font-size: 14px;
		font-weight: 700;
		text-transform: uppercase;
	}

	h1 {
		margin: 0;
		font-size: 40px;
		line-height: 1.1;
	}

	.summary {
		margin: 16px 0 28px;
		color: #555;
		font-size: 16px;
		line-height: 1.5;
	}

	.button {
		display: inline-flex;
		align-items: center;
		justify-content: center;
		min-height: 44px;
		padding: 0 18px;
		border-radius: 6px;
		background: #1f2937;
		color: #fff;
		font-weight: 700;
		text-decoration: none;
	}
</style>

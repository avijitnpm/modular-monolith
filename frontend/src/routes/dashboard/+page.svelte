<script lang="ts">
	import { onMount } from 'svelte';

	type User = {
		subject: string;
		email?: string;
		email_verified?: boolean;
		preferred_username?: string;
		name?: string;
		given_name?: string;
		family_name?: string;
		locale?: string;
		organization_id?: string;
		roles?: string[];
		raw_claims?: Record<string, unknown>;
	};

	let loading = true;
	let user: User | null = null;

	onMount(async () => {
		const response = await fetch('/api/v1/auth/me');

		if (response.status === 401) {
			window.location.href = '/api/v1/auth/login';
			return;
		}

		if (!response.ok) {
			loading = false;
			return;
		}

		const body = await response.json();
		user = body.data.user;
		loading = false;
	});

	async function logout() {
		await fetch('/api/v1/auth/logout', {
			method: 'POST'
		});

		window.location.href = '/';
	}

	function userFields(user: User) {
		return [
			['Subject', user.subject],
			['Email', user.email],
			['Email verified', user.email_verified ? 'Yes' : ''],
			['Preferred username', user.preferred_username],
			['Name', user.name],
			['Given name', user.given_name],
			['Family name', user.family_name],
			['Locale', user.locale],
			['Organization ID', user.organization_id]
		].filter(([, value]) => value);
	}

	function rawClaims(user: User) {
		return user.raw_claims ? Object.entries(user.raw_claims).sort(([a], [b]) => a.localeCompare(b)) : [];
	}
</script>

<main>
	<section>
		<header>
			<div>
				<p class="eyebrow">Dashboard</p>
				<h1>Account</h1>
			</div>

			<button type="button" on:click={logout}>Logout</button>
		</header>

		{#if loading}
			<p class="muted">Loading session...</p>
		{:else if user}
			<dl>
				{#each userFields(user) as [label, value]}
					<div>
						<dt>{label}</dt>
						<dd>{value}</dd>
					</div>
				{/each}
			</dl>

			{#if user.roles?.length}
				<section class="detail-section">
					<h2>Roles</h2>
					<ul>
						{#each user.roles as role}
							<li>{role}</li>
						{/each}
					</ul>
				</section>
			{/if}

			{#if rawClaims(user).length}
				<section class="detail-section">
					<h2>Raw claims</h2>
					<dl>
						{#each rawClaims(user) as [key, value]}
							<div>
								<dt>{key}</dt>
								<dd>{JSON.stringify(value)}</dd>
							</div>
						{/each}
					</dl>
				</section>
			{/if}
		{:else}
			<p class="muted">Unable to load session.</p>
		{/if}
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
		padding: 32px;
		box-sizing: border-box;
	}

	section {
		max-width: 760px;
		margin: 0 auto;
	}

	header {
		display: flex;
		align-items: center;
		justify-content: space-between;
		gap: 16px;
		margin-bottom: 32px;
	}

	.eyebrow {
		margin: 0 0 8px;
		color: #5f6f52;
		font-size: 13px;
		font-weight: 700;
		text-transform: uppercase;
	}

	h1 {
		margin: 0;
		font-size: 36px;
		line-height: 1.1;
	}

	button {
		min-height: 40px;
		padding: 0 14px;
		border: 0;
		border-radius: 6px;
		background: #1f2937;
		color: #fff;
		font: inherit;
		font-weight: 700;
		cursor: pointer;
	}

	dl {
		display: grid;
		gap: 12px;
		margin: 0;
	}

	.detail-section {
		margin-top: 32px;
	}

	h2 {
		margin: 0 0 16px;
		font-size: 20px;
		line-height: 1.2;
	}

	ul {
		display: flex;
		flex-wrap: wrap;
		gap: 8px;
		margin: 0;
		padding: 0;
		list-style: none;
	}

	li {
		padding: 6px 10px;
		border: 1px solid #d0d0c8;
		border-radius: 6px;
		background: #fff;
	}

	dl > div {
		display: grid;
		grid-template-columns: 190px 1fr;
		gap: 16px;
		padding: 16px 0;
		border-top: 1px solid #deded7;
	}

	dt {
		color: #555;
		font-weight: 700;
	}

	dd {
		margin: 0;
		overflow-wrap: anywhere;
	}

	.muted {
		color: #555;
	}

	@media (max-width: 640px) {
		main {
			padding: 24px;
		}

		header,
		dl > div {
			display: grid;
		}

		dl > div {
			grid-template-columns: 1fr;
			gap: 6px;
		}
	}
</style>

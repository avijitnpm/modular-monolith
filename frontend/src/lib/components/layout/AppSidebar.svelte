<script lang="ts">
	import {
		Sidebar,
		SidebarContent,
		SidebarFooter,
		SidebarGroup,
		SidebarGroupContent,
		SidebarGroupLabel,
		SidebarHeader,
		SidebarMenu,
		SidebarMenuButton,
		SidebarMenuItem
	} from '$lib/components/ui/sidebar/index.js';
	import { LayoutDashboard, CreditCard, Users, Shield, Settings } from '@lucide/svelte';
	import { page } from '$app/state';
	import { resolve } from '$app/paths';

	const navItems = [
		{ title: 'Dashboard', href: '/dashboard', icon: LayoutDashboard },
		{ title: 'Billing', href: '/billing', icon: CreditCard },
		{ title: 'Users', href: '/users', icon: Users },
		{ title: 'Roles', href: '/roles', icon: Shield },
		{ title: 'Settings', href: '/settings', icon: Settings }
	] as const;
</script>

<Sidebar>
	<SidebarHeader>
		<div class="flex items-center gap-2 px-2 py-1">
			<div
				class="bg-primary text-primary-foreground flex size-7 items-center justify-center rounded-md text-sm font-bold"
			>
				M
			</div>
			<span class="text-sm font-semibold">Platform</span>
		</div>
	</SidebarHeader>
	<SidebarContent>
		<SidebarGroup>
			<SidebarGroupLabel>Navigation</SidebarGroupLabel>
			<SidebarGroupContent>
				<SidebarMenu>
					{#each navItems as item (item.href)}
						<SidebarMenuItem>
							<SidebarMenuButton isActive={page.url.pathname.startsWith(item.href)}>
								{#snippet child({ props })}
									<a href={resolve(item.href)} {...props}>
										<item.icon />
										<span>{item.title}</span>
									</a>
								{/snippet}
							</SidebarMenuButton>
						</SidebarMenuItem>
					{/each}
				</SidebarMenu>
			</SidebarGroupContent>
		</SidebarGroup>
	</SidebarContent>
	<SidebarFooter>
		<div class="text-muted-foreground px-2 py-1 text-xs">v0.1.0</div>
	</SidebarFooter>
</Sidebar>

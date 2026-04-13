<script>
  import { onMount } from 'svelte';
  import { authToken } from '$lib/api.js';
  import { goto } from '$app/navigation';
  import { page } from '$app/stores';
  import { get } from 'svelte/store';

  onMount(() => {
    if (!get(authToken)) goto('/login');
  });

  function logout() {
    authToken.clear();
    goto('/login');
  }

  const menu = [
    { href: '/admin', label: 'Dashboard', icon: '📊' },
    { href: '/admin/products', label: 'Produk', icon: '📦' },
    { href: '/admin/orders', label: 'Pesanan', icon: '🧾' },
    { href: '/admin/payment', label: 'Pembayaran', icon: '💳' },
    { href: '/admin/scripts', label: 'Script Logs', icon: '⚙️' },
    { href: '/admin/providers', label: 'Provider Stok', icon: '🔌' },
  ];

  $: current = $page.url.pathname;
</script>

<div class="admin-shell">
  <!-- Top bar -->
  <header class="admin-header">
    <a href="/" class="brand">
      <span class="brand-logo">🛍</span>
      <span>DigiStore</span>
    </a>
    <span style="font-size:12px;color:var(--text-muted)">Admin Panel</span>
    <div style="display:flex;gap:8px;margin-left:auto">
      <a href="/" class="btn btn-sm">← Toko</a>
      <button class="btn btn-sm btn-danger" on:click={logout}>Logout</button>
    </div>
  </header>

  <div class="admin-body">
    <!-- Sidebar -->
    <aside class="sidebar">
      <div class="sidebar-label">Menu</div>
      {#each menu as item}
        <a
          href={item.href}
          class="sidebar-item {current === item.href ? 'active' : ''}"
        >
          <span class="sidebar-icon">{item.icon}</span>
          {item.label}
        </a>
      {/each}
    </aside>

    <!-- Main content -->
    <main class="admin-main">
      <slot />
    </main>
  </div>
</div>

<style>
.admin-shell { min-height: 100vh; display: flex; flex-direction: column; }
.admin-header {
  position: sticky; top: 0; z-index: 100;
  background: #fff; border-bottom: 0.5px solid var(--border);
  padding: 0 1.5rem;
  height: 54px; display: flex; align-items: center; gap: 10px;
}
.brand { display: flex; align-items: center; gap: 8px; font-weight: 500; font-size: 15px; }
.brand-logo {
  background: #0d5fa8; border-radius: 8px;
  width: 28px; height: 28px; display: flex; align-items: center; justify-content: center;
  font-size: 14px;
}
.admin-body { display: flex; flex: 1; }
.sidebar {
  width: 190px; flex-shrink: 0;
  background: #fff; border-right: 0.5px solid var(--border);
  padding: 1rem 0.75rem;
  position: sticky; top: 54px; height: calc(100vh - 54px);
  overflow-y: auto;
}
.sidebar-label {
  font-size: 10.5px; text-transform: uppercase; letter-spacing: 0.6px;
  color: var(--text-hint); padding: 0 10px; margin-bottom: 8px;
}
.sidebar-item {
  display: flex; align-items: center; gap: 8px;
  padding: 8px 10px; border-radius: var(--radius);
  font-size: 13.5px; color: var(--text);
  transition: background 0.15s;
  margin-bottom: 2px;
}
.sidebar-item:hover { background: #f4f4f2; }
.sidebar-item.active { background: #E6F1FB; color: #185FA5; font-weight: 500; }
.sidebar-icon { font-size: 14px; }
.admin-main {
  flex: 1; min-width: 0;
  padding: 1.5rem 1.25rem;
  max-width: 1000px;
}
</style>

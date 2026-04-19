<script>
  import { onMount } from 'svelte';
  import { authToken } from '$lib/api.js';
  import { goto } from '$app/navigation';
  import { page } from '$app/stores';
  import { get } from 'svelte/store';
  import ThemeToggle from '$lib/ThemeToggle.svelte';

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
    { href: '/admin/providers', label: 'Pull Provider', icon: '🔌' },
    { href: '/admin/koalastore', label: 'KoalaStore', icon: '🐨' },
    { href: '/admin/contact', label: 'Kontak & Support', icon: '📞' },
  ];

  $: current = $page.url.pathname;
</script>

<div class="admin-shell">
  <!-- Top bar -->
  <header class="admin-header">
    <div class="brand-wrap">
      <a href="/" class="brand">
        <span class="brand-logo">🛍</span>
        <span>DigiStore</span>
      </a>
      <span class="brand-sub">Modern digital store dashboard</span>
    </div>
    <div class="header-actions">
      <a href="/" class="btn btn-sm">Toko</a>
      <ThemeToggle />
      <button class="btn btn-sm btn-danger" on:click={logout}>Logout</button>
    </div>
  </header>

  <div class="admin-body">
    <!-- Sidebar -->
    <aside class="sidebar">
      <div class="sidebar-label">Workspace</div>
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
  background: color-mix(in srgb, var(--surface) 84%, transparent);
  backdrop-filter: blur(18px);
  border-bottom: 1px solid var(--border);
  padding: 0 1.5rem;
  height: 68px; display: flex; align-items: center; gap: 12px;
}
.brand-wrap { display:flex; flex-direction:column; gap:2px; }
.brand { display: flex; align-items: center; gap: 8px; font-weight: 700; font-size: 15px; letter-spacing:-0.02em; }
.brand-logo {
  background: linear-gradient(135deg, var(--primary), var(--primary-2));
  border-radius: 10px;
  width: 30px; height: 30px; display: flex; align-items: center; justify-content: center;
  font-size: 14px; box-shadow: 0 10px 20px rgba(21,93,252,0.2);
  color: var(--primary-fg);
}
.brand-sub { font-size: 11.5px; color: var(--text-muted); margin-left: 38px; margin-top: -2px; }
.header-actions { display:flex;gap:8px;margin-left:auto; }
.admin-body { display: flex; flex: 1; }
.sidebar {
  width: 214px; flex-shrink: 0;
  background: color-mix(in srgb, var(--surface) 84%, transparent);
  backdrop-filter: blur(18px);
  border-right: 1px solid var(--border);
  padding: 1rem 0.85rem;
  position: sticky; top: 68px; height: calc(100vh - 68px);
  overflow-y: auto;
}
.sidebar-label {
  font-size: 10.5px; text-transform: uppercase; letter-spacing: 0.12em;
  color: var(--text-hint); padding: 0 10px; margin-bottom: 8px;
}
.sidebar-item {
  display: flex; align-items: center; gap: 8px;
  padding: 10px 12px; border-radius: 12px;
  font-size: 13.5px; color: var(--text);
  transition: background 0.15s;
  margin-bottom: 2px;
}
.sidebar-item:hover { background: var(--surface-2); }
.sidebar-item.active { background: linear-gradient(135deg, var(--primary-bg), color-mix(in srgb, var(--primary-bg) 70%, var(--surface))); color: var(--primary); font-weight: 600; box-shadow: inset 0 0 0 1px var(--border); }
.sidebar-icon { font-size: 14px; }
.admin-main {
  flex: 1; min-width: 0;
  padding: 1.5rem 1.25rem 2rem;
  max-width: 1180px;
}

@media (max-width: 900px) {
  .admin-body { flex-direction: column; }
  .sidebar {
    width: 100%; height: auto; position: relative; top: 0;
    border-right: none; border-bottom: 1px solid var(--border);
    display:flex; flex-wrap:wrap; gap:8px; align-items:center;
    padding: 0.85rem 1rem;
  }
  .sidebar-label { width: 100%; margin-bottom: 2px; }
  .sidebar-item { margin-bottom: 0; white-space: nowrap; }
  .admin-main { padding: 1rem; }
}

@media (max-width: 640px) {
  .admin-header { height: auto; min-height: 64px; padding: 0.8rem 1rem; flex-wrap: wrap; align-items:flex-start; }
  .brand-wrap { width: 100%; }
  .brand-sub { margin-left: 38px; }
  .header-actions { width: 100%; margin-left: 38px; justify-content: flex-start; flex-wrap: wrap; }
  .sidebar { padding: 0.75rem 1rem; }
  .sidebar-label { display:none; }
  .sidebar-item { font-size: 12.5px; padding: 8px 10px; }
  .admin-main { padding: 0.85rem; }
}
</style>

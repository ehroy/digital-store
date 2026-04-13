<script>
  import { onMount } from 'svelte';
  import { api } from '$lib/api.js';
  import { IDR, fmtDate, STATUS_LABEL } from '$lib/utils.js';

  let data = null;
  let loading = true;
  let error = '';

  onMount(async () => {
    try { data = await api.dashboard(); }
    catch (e) { error = e.message; }
    finally { loading = false; }
  });

  $: stats = data ? [
    { label: 'Total Pemasukan', value: IDR(data.total_revenue), accent: '#0d5fa8' },
    { label: 'Total Pesanan', value: data.total_orders },
    { label: 'Pesanan Lunas', value: data.paid_orders, accent: '#2f5e0f' },
    { label: 'Menunggu Bayar', value: data.pending_orders, accent: data.pending_orders > 0 ? '#854F0B' : undefined },
    { label: 'Produk Aktif', value: data.active_products },
    { label: 'Stok Hampir Habis', value: data.low_stock, accent: data.low_stock > 0 ? '#8c2626' : undefined },
  ] : [];

  $: maxRev = data?.revenue_by_category?.length
    ? Math.max(...data.revenue_by_category.map(c => c.Revenue || c.revenue || 0))
    : 1;
</script>

<svelte:head><title>Dashboard — Digitalkuh Murah Admin</title></svelte:head>

{#if loading}
  <div style="color:var(--text-muted);padding:2rem">Memuat data…</div>
{:else if error}
  <div class="alert-error">{error}</div>
{:else if data}
  <div class="page-header"><h1 class="page-title">Dashboard</h1></div>

  <!-- Stats grid -->
  <div class="stats-grid">
    {#each stats as s}
      <div class="stat-card">
        <div class="stat-label">{s.label}</div>
        <div class="stat-value" style="color:{s.accent || 'var(--text)'}">{s.value}</div>
      </div>
    {/each}
  </div>

  <!-- Revenue by category -->
  {#if data.revenue_by_category?.length}
    <div class="card" style="margin-top:16px">
      <div style="font-weight:500;font-size:14px;margin-bottom:14px">Pendapatan per Kategori</div>
      <div class="rev-bars">
        {#each data.revenue_by_category as row}
          {@const rev = row.Revenue || row.revenue || 0}
          {@const pct = Math.round(rev / maxRev * 100)}
          <div class="rev-row">
            <div class="rev-meta">
              <span>{row.Category || row.category}</span>
              <span style="font-weight:500">{IDR(rev)} <span style="color:var(--text-muted);font-weight:400">({pct}%)</span></span>
            </div>
            <div class="rev-track"><div class="rev-fill" style="width:{pct}%"></div></div>
          </div>
        {/each}
      </div>
    </div>
  {/if}

  <!-- Recent orders -->
  <div class="card" style="margin-top:16px;padding:0;overflow:hidden">
    <div style="padding:1rem 1.25rem;font-weight:500;font-size:14px;border-bottom:0.5px solid var(--border)">
      Pesanan Terbaru
    </div>
    <div style="overflow-x:auto">
      <table class="data-table">
        <thead>
          <tr>
            <th>Invoice</th><th>Produk</th><th>Pembeli</th>
            <th>Total</th><th>Status</th><th>Tanggal</th>
          </tr>
        </thead>
        <tbody>
          {#each data.recent_orders as o}
            <tr>
              <td class="mono">{o.invoice_no}</td>
              <td>{o.product_name}</td>
              <td>{o.buyer_name}</td>
              <td style="font-weight:500">{IDR(o.total)}</td>
              <td><span class="badge badge-{o.status}">{STATUS_LABEL[o.status] || o.status}</span></td>
              <td style="color:var(--text-muted)">{fmtDate(o.created_at)}</td>
            </tr>
          {/each}
        </tbody>
      </table>
    </div>
  </div>
{/if}

<style>
.stats-grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(155px, 1fr));
  gap: 11px; margin-bottom: 16px;
}
.rev-bars { display: flex; flex-direction: column; gap: 12px; }
.rev-row {}
.rev-meta {
  display: flex; justify-content: space-between;
  font-size: 13px; margin-bottom: 5px;
}
.rev-track {
  height: 6px; background: #f0f0ee; border-radius: 99px; overflow: hidden;
}
.rev-fill { height: 100%; background: #0d5fa8; border-radius: 99px; transition: width 0.4s; }
</style>

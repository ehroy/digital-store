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
     { label: 'Total Pemasukan', value: IDR(data.total_revenue), accent: 'var(--primary)', hint: 'Akumulasi omzet' },
    { label: 'Total Pesanan', value: data.total_orders, hint: 'Semua transaksi' },
     { label: 'Pesanan Lunas', value: data.paid_orders, accent: 'var(--success-fg)', hint: 'Siap diproses' },
     { label: 'Menunggu Bayar', value: data.pending_orders, accent: data.pending_orders > 0 ? 'var(--warning-fg)' : undefined, hint: 'Butuh pembayaran' },
    { label: 'Produk Aktif', value: data.active_products, hint: 'Siap dijual' },
     { label: 'Stok Hampir Habis', value: data.low_stock, accent: data.low_stock > 0 ? 'var(--danger-fg)' : undefined, hint: 'Perlu dicek' },
  ] : [];

  $: maxRev = data?.revenue_by_category?.length
    ? Math.max(...data.revenue_by_category.map(c => c.Revenue || c.revenue || 0))
    : 1;
</script>

<svelte:head><title>Dashboard — DigiStore Admin</title></svelte:head>

{#if loading}
  <div style="color:var(--text-muted);padding:2rem">Memuat data…</div>
{:else if error}
  <div class="alert-error">{error}</div>
{:else if data}
  <div class="dash-hero card">
    <div>
      <div class="eyebrow">Admin Dashboard</div>
      <h1 class="hero-title">Ringkasan store digital yang bersih dan cepat dibaca.</h1>
      <p class="hero-desc">Pantau omzet, status pesanan, dan stok dari satu layar dengan tampilan minimal modern.</p>
    </div>
    <div class="hero-card-mini">
      <div class="mini-label">Hari ini</div>
      <div class="mini-value">{data.total_orders} pesanan</div>
      <div class="mini-sub">{IDR(data.total_revenue)} total pemasukan</div>
    </div>
  </div>

  <!-- Stats grid -->
  <div class="stats-grid">
    {#each stats as s}
      <div class="stat-card stat-card-hero">
        <div class="stat-label">{s.label}</div>
        <div class="stat-value" style="color:{s.accent || 'var(--text)'}">{s.value}</div>
        <div class="stat-hint">{s.hint}</div>
      </div>
    {/each}
  </div>

  <!-- Revenue by category -->
  {#if data.revenue_by_category?.length}
    <div class="card" style="margin-top:16px">
      <div class="section-head">
        <div>
          <div class="section-title">Pendapatan per Kategori</div>
          <div class="section-sub">Komposisi kategori yang paling berkontribusi.</div>
        </div>
      </div>
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
    <div class="section-head section-head-tight">
      <div>
        <div class="section-title">Pesanan Terbaru</div>
        <div class="section-sub">Update transaksi terakhir di toko.</div>
      </div>
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
  grid-template-columns: repeat(auto-fill, minmax(180px, 1fr));
  gap: 12px; margin-bottom: 16px;
}
.rev-bars { display: flex; flex-direction: column; gap: 12px; }
.dash-hero {
  display: grid;
  grid-template-columns: minmax(0, 1.8fr) minmax(220px, 0.8fr);
  gap: 16px;
  margin-bottom: 16px;
  padding: 1.35rem 1.35rem 1.25rem;
    background:
     radial-gradient(circle at top right, rgba(21,93,252,0.16), transparent 32%),
     linear-gradient(180deg, var(--surface) 0%, var(--surface-2) 100%);
}
.eyebrow { font-size: 11px; text-transform: uppercase; letter-spacing: 0.12em; color: var(--primary); font-weight: 600; margin-bottom: 8px; }
.hero-title { font-size: 26px; line-height: 1.15; letter-spacing: -0.03em; margin-bottom: 8px; font-weight: 700; }
.hero-desc { color: var(--text-muted); max-width: 58ch; }
.hero-card-mini {
  align-self: stretch; border-radius: 16px; padding: 1rem 1rem 0.95rem;
  background: var(--surface-2); border: 1px solid var(--border);
  display:flex; flex-direction:column; justify-content:flex-end; gap:4px;
}
.mini-label { font-size: 11px; text-transform: uppercase; letter-spacing: 0.12em; color: var(--text-hint); }
.mini-value { font-size: 21px; font-weight: 700; letter-spacing: -0.03em; }
.mini-sub { color: var(--text-muted); font-size: 13px; }
.section-head {
  padding: 1rem 1.25rem;
  border-bottom: 1px solid var(--border);
}
.section-head-tight { border-bottom: 1px solid var(--border); }
.section-title { font-size: 14px; font-weight: 600; letter-spacing: -0.01em; }
.section-sub { font-size: 12.5px; color: var(--text-muted); margin-top: 2px; }
.rev-meta {
  display: flex; justify-content: space-between;
  font-size: 13px; margin-bottom: 5px;
}
.rev-track {
  height: 7px; background: var(--surface-2); border-radius: 99px; overflow: hidden;
}
.rev-fill { height: 100%; background: linear-gradient(90deg, var(--primary), var(--primary-2)); border-radius: 99px; transition: width 0.4s; }
.stat-card-hero { position: relative; overflow: hidden; }
.stat-card-hero::after {
  content: '';
  position: absolute; inset: auto -20px -18px auto;
  width: 74px; height: 74px; border-radius: 999px;
  background: radial-gradient(circle, rgba(21,93,252,0.12), transparent 70%);
}
.stat-hint { margin-top: 6px; font-size: 11.5px; color: var(--text-muted); }
@media (max-width: 800px) {
  .dash-hero { grid-template-columns: 1fr; }
}
</style>

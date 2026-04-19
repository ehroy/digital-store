<script>
  import { onMount, onDestroy } from 'svelte';
  import { api } from '$lib/api.js';
  import { IDR } from '$lib/utils.js';
  import ThemeToggle from '$lib/ThemeToggle.svelte';
  import BuyModal from './BuyModal.svelte';
  import CheckoutModal from './CheckoutModal.svelte';
  import InvoiceModal from './InvoiceModal.svelte';

  let products = [];
  let loading = true;
  let loadingMore = false;
  let hasMore = false;
  let totalProducts = 0;
  let page = 1;
  let perPage = 10;
  let search = '';
  let activeCategory = 'all';
  let categories = ['all'];
  let contact = null;
  let loadMoreSentinel;
  let observer;
  let searchTimer;
  let querySeq = 0;

  let buyProduct = null;
  let checkoutData = null;
  let invoiceOrder = null;

  function buildQuery(nextPage) {
    const params = new URLSearchParams();
    params.set('sort', 'terlaris');
    params.set('page', String(nextPage));
    params.set('per_page', String(perPage));
    if (search.trim()) params.set('search', search.trim());
    if (activeCategory !== 'all') params.set('category', activeCategory);
    return `?${params.toString()}`;
  }

  async function loadPage(nextPage = 1, replace = true) {
    const seq = ++querySeq;
    if (replace) loading = true;
    else loadingMore = true;
    try {
      const res = await api.getProducts(buildQuery(nextPage));
      if (seq !== querySeq) return;
      const items = Array.isArray(res) ? res : (res.items || []);
      if (replace) products = items;
      else products = [...products, ...items];
      page = Array.isArray(res) ? 1 : (res.page || nextPage);
      totalProducts = Array.isArray(res) ? items.length : (res.total || items.length);
      hasMore = Array.isArray(res) ? false : page < (res.total_pages || 1) && items.length > 0;
      if (replace && categories.length === 1 && Array.isArray(res) === false && Array.isArray(res.categories) && res.categories.length > 0) {
        categories = ['all', ...res.categories];
      }
    } finally {
      loading = false;
      loadingMore = false;
    }
  }

  function resetAndLoad() {
    loadPage(1, true);
  }

  function scheduleReload() {
    clearTimeout(searchTimer);
    searchTimer = setTimeout(() => resetAndLoad(), 250);
  }

  onMount(async () => {
    api.getContact().then(c => contact = c).catch(() => {});
    try {
      await loadPage(1, true);
      observer = new IntersectionObserver((entries) => {
        if (entries[0]?.isIntersecting && hasMore && !loadingMore) {
          loadPage(page + 1, false);
        }
      }, { rootMargin: '280px 0px' });
      if (loadMoreSentinel) observer.observe(loadMoreSentinel);
    } finally { loading = false; }
  });

  onDestroy(() => {
    clearTimeout(searchTimer);
    observer?.disconnect();
  });

  $: filtered = products;

  // Status stok untuk setiap produk
  function stockInfo(p) {
    if (p.type === 'script') return { label: 'Tersedia', color: '#2f5e0f', canBuy: true };
    if (p.type === 'stock') {
      if (p.available_stock === 0) return { label: 'Stok Habis', color: '#8c2626', canBuy: false };
      if (p.available_stock < 5)   return { label: `Sisa ${p.available_stock}`, color: '#854F0B', canBuy: true };
      return { label: 'Tersedia', color: '#2f5e0f', canBuy: true };
    }
    if (p.type === 'provider') {
      const totalStock = Number(p.available_stock) || 0;
      const variants = Array.isArray(p.variants) ? p.variants : [];
      const availableVariants = variants.filter(v => v.stock_status !== 'out_of_stock');
      const variantStockTotal = variants.reduce((sum, v) => sum + (Number(v.available_stock) || 0), 0);
      if (variants.length > 0) {
        if (availableVariants.length === 0) {
          return { label: 'Habis', color: '#8c2626', canBuy: false };
        }
        return { label: `${availableVariants.length} varian aktif`, color: '#2f5e0f', canBuy: true, stock: variantStockTotal };
      }
      switch (p.provider_status) {
        case 'available':
          return { label: totalStock > 0 ? `Stok ${totalStock}` : 'Cek Stok...', color: '#2f5e0f', canBuy: totalStock > 0, stock: totalStock };
        case 'out_of_stock':
          return { label: 'Habis di Provider', color: '#8c2626', canBuy: false };
        case 'manual':
          return totalStock > 0
            ? { label: `Stok ${totalStock}`, color: '#2f5e0f', canBuy: true, stock: totalStock }
            : { label: 'Proses Manual', color: '#854F0B', canBuy: true };
        default:
          return totalStock > 0
            ? { label: `Stok ${totalStock}`, color: '#2f5e0f', canBuy: true, stock: totalStock }
            : { label: 'Cek Stok...', color: '#999', canBuy: false };
      }
    }
    return { label: '—', color: '#999', canBuy: false };
  }

  $: waHref = contact?.whatsapp
    ? `https://wa.me/${contact.whatsapp.replace(/\D/g, '')}`
    : '';
</script>

<svelte:head><title>Digital Murah — Produk Digital</title></svelte:head>

<nav class="store-nav">
  <div class="nav-inner">
    <a href="/" class="nav-brand">
      <span class="nav-logo">🛍</span>
      <span>Digital Murah</span>
    </a>
    <span class="nav-tag">Digital goods, clean checkout</span>
    <div class="nav-actions">
      <a href="/cek-invoice" class="btn btn-sm">📋 Cek Invoice</a>
      <a href="/komplain" class="btn btn-sm">🎧 Bantuan</a>

    </div>
  </div>
</nav>

<ThemeToggle floating />

<div class="hero">
  <div class="hero-panel">
    <div class="hero-orb hero-orb-a"></div>
    <div class="hero-orb hero-orb-b"></div>
    <div class="hero-copy">
      <div class="hero-kicker">Digital Murah</div>
      <h1>Belanja produk digital jadi lebih rapi, cepat, dan meyakinkan.</h1>
      <p>Temukan produk, cek status, lalu bayar dengan QRIS tanpa alur yang bertele-tele. Semua dibuat lebih jelas untuk user.</p>
      <div class="hero-actions">
        <a href="#produk" class="btn btn-primary">Lihat produk</a>
        <a href="/cek-invoice" class="btn">Cek invoice</a>
      </div>
      <div class="hero-chips">
        <span>QRIS aktif</span>
        <span>Status transparan</span>
        <span>Support WhatsApp</span>
      </div>
    </div>
    <div class="hero-aside">
        <div class="aside-label">Pilihan cepat</div>
        <div class="aside-value">Checkout mudah semua payment dengan QRIS</div>
      <div class="aside-sub">Invoice, status pembayaran, dan komplain semuanya rapi di satu tempat.</div>
      <div class="aside-stats">
        <div>
          <strong>1x</strong>
          <span>alur bayar</span>
        </div>
        <div>
          <strong>Live</strong>
          <span>cek status</span>
        </div>
        <div>
          <strong>Rapi</strong>
          <span>tampilan toko</span>
        </div>
      </div>
    </div>
  </div>
</div>

<div class="store-container" id="produk">
  <div class="filter-row">
    <input class="input" placeholder="🔍  Cari produk..." bind:value={search} on:input={scheduleReload} />
    <div class="cat-pills">
      {#each categories as cat}
        <button class="btn btn-sm {activeCategory===cat?'btn-primary':''}"
          on:click={() => { activeCategory = cat; resetAndLoad(); }}>
          {cat === 'all' ? 'Semua' : cat}
        </button>
      {/each}
    </div>
  </div>

  {#if loading}
    <div class="empty-state">Memuat produk…</div>
  {:else if filtered.length === 0}
    <div class="empty-state">Produk tidak ditemukan.</div>
  {:else}
    <div class="product-grid">
      {#each filtered as product, index (product.id)}
        {@const stock = stockInfo(product)}
        <div class="product-card {!stock.canBuy?'out-of-stock':''}" style="animation-delay:{Math.min(index % perPage, 9) * 18}ms">
          {#if product.image_url}
            <div class="product-img-wrap">
              <img src={product.image_url} alt={product.name} class="product-img" loading="lazy" />
            </div>
          {:else}
            <div class="product-icon">{product.icon}</div>
          {/if}

          <div class="product-body">
            <div style="margin-bottom:6px;display:flex;gap:6px;flex-wrap:wrap">
              <span class="badge badge-stock">{product.category}</span>
              {#if product.is_popular}
                <span class="badge" style="background:#FFF7ED;color:#B45309">Terlaris</span>
              {/if}
        
            
              {#if product.type === 'script'}
                <span class="badge" style="background:#EEEDFE;color:#534AB7">Jasa</span>
              {/if}
            </div>
            <div class="product-name">{product.name}</div>
            <div class="product-desc">
              {(product.description||'').length > 90
                ? product.description.slice(0, 87) + '…'
                : product.description}
            </div>
          </div>

          <div class="product-footer">
            <div class="product-price">
              {#if product.type === 'provider' && Array.isArray(product.variants) && product.variants.length > 0}
                Mulai {IDR(product.price)}
              {:else}
                {IDR(product.price)}
              {/if}
            </div>
            <div class="stock-badge" style="color:{stock.color};font-weight:600">
              {#if product.type === 'provider' && product.provider_status === 'available'}
                <span class="stock-dot" style="background:{stock.color}"></span>
              {/if}
              {stock.label}
            </div>
            {#if stock.detail}
              <div style="grid-column:1/-1;font-size:11px;color:var(--text-muted);margin-top:-2px">
                {stock.detail}
              </div>
            {/if}
          </div>

          {#if product.type === 'provider' && Array.isArray(product.variants) && product.variants.length > 0}
            <div class="variant-badges">
              {#each product.variants.slice(0, 1) as variant}
                <span class="variant-pill">
                  {variant.variant_name || 'Varian'} · {IDR(variant.price)}
                </span>
              {/each}
              {#if product.variants.length > 1}
                <span class="variant-pill muted">+{product.variants.length - 1} varian</span>
              {/if}
            </div>
          {/if}

          <div class="product-actions">
            <button class="btn btn-primary"
              disabled={!stock.canBuy || !product.active}
              on:click={() => buyProduct = product}>
              {!stock.canBuy ? 'Tidak Tersedia' : 'Beli Sekarang'}
            </button>
            <a href="/product/{product.id}" class="btn detail-btn">
              Detail
            </a>
          </div>
        </div>
      {/each}
    </div>
    <div bind:this={loadMoreSentinel} style="height:1px"></div>
    <div class="load-more-meta">
      <span>{products.length} dari {totalProducts} produk</span>
      {#if hasMore}
        <button class="btn btn-sm" on:click={() => loadPage(page + 1, false)} disabled={loadingMore}>
          {loadingMore ? 'Memuat…' : 'Muat 10 lagi'}
        </button>
      {/if}
    </div>
    {#if loadingMore}
      <div class="load-more-state">Memuat produk berikutnya…</div>
    {:else if hasMore}
      <div class="load-more-state muted">Scroll untuk memuat produk berikutnya</div>
    {/if}
  {/if}

    <div class="store-footer">
      <p>Sudah beli? <a href="/cek-invoice" style="color:var(--primary);font-weight:500">Cek status &amp; download produk →</a></p>
    </div>
</div>

<!-- Floating WA button -->
{#if contact?.whatsapp}
  <a href={waHref} target="_blank" rel="noopener" class="wa-float"
    title={contact.whatsapp_label || 'Hubungi CS'}>
    <svg viewBox="0 0 24 24" width="22" height="22" fill="white">
      <path d="M17.472 14.382c-.297-.149-1.758-.867-2.03-.967-.273-.099-.471-.148-.67.15-.197.297-.767.966-.94 1.164-.173.199-.347.223-.644.075-.297-.15-1.255-.463-2.39-1.475-.883-.788-1.48-1.761-1.653-2.059-.173-.297-.018-.458.13-.606.134-.133.298-.347.446-.52.149-.174.198-.298.298-.497.099-.198.05-.371-.025-.52-.075-.149-.669-1.612-.916-2.207-.242-.579-.487-.5-.669-.51-.173-.008-.371-.01-.57-.01-.198 0-.52.074-.792.372-.272.297-1.04 1.016-1.04 2.479 0 1.462 1.065 2.875 1.213 3.074.149.198 2.096 3.2 5.077 4.487.709.306 1.262.489 1.694.625.712.227 1.36.195 1.871.118.571-.085 1.758-.719 2.006-1.413.248-.694.248-1.289.173-1.413-.074-.124-.272-.198-.57-.347m-5.421 7.403h-.004a9.87 9.87 0 01-5.031-1.378l-.361-.214-3.741.982.998-3.648-.235-.374a9.86 9.86 0 01-1.51-5.26c.001-5.45 4.436-9.884 9.888-9.884 2.64 0 5.122 1.03 6.988 2.898a9.825 9.825 0 012.893 6.994c-.003 5.45-4.437 9.884-9.885 9.884m8.413-18.297A11.815 11.815 0 0012.05 0C5.495 0 .16 5.335.157 11.892c0 2.096.547 4.142 1.588 5.945L.057 24l6.305-1.654a11.882 11.882 0 005.683 1.448h.005c6.554 0 11.89-5.335 11.893-11.893a11.821 11.821 0 00-3.48-8.413z"/>
    </svg>
    <span>{contact.whatsapp_label || 'Hubungi CS'}</span>
  </a>
{/if}

{#if buyProduct}
  <BuyModal product={buyProduct} on:close={() => buyProduct = null}
    on:checkout={(e) => { checkoutData = e.detail; buyProduct = null; }} />
{/if}
{#if checkoutData}
  <CheckoutModal data={checkoutData} on:close={() => checkoutData = null}
    on:done={(e) => { checkoutData = null; invoiceOrder = e.detail; }} />
{/if}
{#if invoiceOrder}
  <InvoiceModal order={invoiceOrder} on:close={() => invoiceOrder = null} />
{/if}

<style>
.store-nav { position:sticky;top:0;z-index:100;background:var(--surface);border-bottom:0.5px solid var(--border);padding:0 1.5rem; backdrop-filter: blur(14px); }
.nav-inner { max-width:1100px;margin:0 auto;min-height:64px;display:flex;align-items:center;gap:10px;flex-wrap:wrap;padding:8px 0; }
.nav-brand { display:flex;align-items:center;gap:8px;font-weight:700;font-size:15.5px; letter-spacing:-0.02em; }
.nav-logo { background:linear-gradient(135deg,var(--primary),var(--primary-2));border-radius:10px;width:30px;height:30px;display:flex;align-items:center;justify-content:center;font-size:14px; box-shadow:0 10px 20px rgba(21,93,252,0.18); color:var(--primary-fg); }
.nav-tag { font-size:11.5px;color:var(--text-muted);padding:6px 10px;border-radius:999px;background:var(--surface-2); border:1px solid var(--border); }
.nav-actions { display:flex;gap:8px;margin-left:auto;flex-wrap:wrap;justify-content:flex-end;min-width:0; }
.nav-actions :global(.btn) { white-space:nowrap; }
.hero { padding:2.1rem 1rem 1.2rem; }
.hero-panel {
  position:relative;overflow:hidden;
  max-width:1100px;margin:0 auto;
  display:grid;grid-template-columns:minmax(0,1.5fr) minmax(220px,0.7fr);gap:14px;
  background:
    radial-gradient(circle at top left, rgba(96,165,250,0.16), transparent 34%),
    linear-gradient(180deg,var(--surface),var(--surface-2));
  border:1px solid var(--border);border-radius:24px;padding:1.35rem;box-shadow:var(--shadow);
}
.hero-orb { position:absolute;border-radius:999px;filter:blur(12px);pointer-events:none; }
.hero-orb-a { top:-36px; right:12%; width:140px; height:140px; background:rgba(37,99,235,0.08); }
.hero-orb-b { bottom:-28px; left:38%; width:92px; height:92px; background:rgba(14,165,233,0.10); }
.hero-copy, .hero-aside { position:relative; z-index:1; }
.hero-copy h1 { font-size: clamp(28px, 4vw, 42px); line-height:1.05; letter-spacing:-0.04em; margin-bottom:10px; max-width:12ch; }
.hero-copy p { color:var(--text-muted);font-size:14.5px;max-width:58ch; }
.hero-kicker { display:inline-flex;align-items:center;gap:6px;font-size:11px;text-transform:uppercase;letter-spacing:0.14em;color:var(--primary);font-weight:700;margin-bottom:10px; }
.hero-actions { display:flex;gap:10px;flex-wrap:wrap;margin-top:16px; }
.hero-chips { display:flex;gap:8px;flex-wrap:wrap;margin-top:14px; }
.hero-chips span {
  font-size:12px;padding:6px 10px;border-radius:999px;
  background:rgba(21,93,252,0.08);color:#0f4fd6;border:1px solid rgba(21,93,252,0.10);
}
.hero-aside {
  border-radius:18px;padding:1rem 1rem 0.95rem;
  background:linear-gradient(180deg,var(--primary-bg),var(--surface));
  border:1px solid rgba(21,93,252,0.12);
  display:flex;flex-direction:column;justify-content:flex-end;gap:6px;
}
.aside-label { font-size:11px;text-transform:uppercase;letter-spacing:0.12em;color:#3b82f6;font-weight:700; }
.aside-value { font-size:22px;font-weight:800;letter-spacing:-0.03em; }
.aside-sub { color:var(--text-muted);font-size:13px;line-height:1.6; }
.aside-stats { display:grid;grid-template-columns:repeat(3,1fr);gap:8px;margin-top:12px; }
.aside-stats div { background:var(--surface);border:1px solid var(--border);border-radius:14px;padding:10px 8px;display:flex;flex-direction:column;gap:2px;align-items:center;text-align:center; }
.aside-stats strong { font-size:13px;color:#0f4fd6; }
.aside-stats span { font-size:11px;color:var(--text-muted); }
.store-container { max-width:1100px;margin:0 auto;padding:1.25rem 1rem 3rem; }
.filter-row { display:flex;gap:10px;margin-bottom:1.5rem;flex-wrap:wrap;align-items:center; }
.filter-row .input { flex:1;min-width:200px; }
.cat-pills { display:flex;gap:6px;flex-wrap:wrap; }
.product-grid { display:grid;grid-template-columns:repeat(auto-fill,minmax(245px,1fr));gap:14px; }
.product-card { background:var(--surface);border:1px solid var(--border);border-radius:var(--radius-lg);padding:1.15rem;display:flex;flex-direction:column;gap:12px;transition:transform 180ms cubic-bezier(.2,.8,.2,1), box-shadow 180ms cubic-bezier(.2,.8,.2,1), border-color 180ms cubic-bezier(.2,.8,.2,1); box-shadow:0 10px 26px rgba(15,23,42,0.05); will-change: transform; animation: card-enter 320ms cubic-bezier(.2,.8,.2,1) both; }
.product-card:hover:not(.out-of-stock) { box-shadow:0 18px 40px rgba(15,23,42,0.10); transform:translateY(-4px) scale(1.012); border-color:rgba(21,93,252,0.15); animation: card-bounce 320ms ease-out; }
.out-of-stock { opacity:0.7; }
.product-img-wrap { border-radius:18px;overflow:hidden;background:var(--surface-2);aspect-ratio:16/16; border:1px solid var(--border); }
.product-img { width:100%;height:100%;object-fit:cover;display:block;transition:transform 0.3s; }
.product-card:hover .product-img { transform:scale(1.04); }
.product-icon { font-size:40px;text-align:center;background:linear-gradient(180deg,var(--surface-2),var(--bg-elevated));border-radius:18px;padding:1rem;border:1px solid var(--border); }
.product-body { flex:1; }
.product-name { font-weight:700;font-size:14.8px;line-height:1.4;margin-bottom:5px;letter-spacing:-0.01em; }
.product-desc { font-size:12.5px;color:var(--text-muted);line-height:1.6; }
.product-footer { display:flex;justify-content:space-between;align-items:center;padding-top:8px;border-top:1px solid var(--border); }
.product-price { font-weight:700;color:var(--primary);font-size:16px; }
.stock-badge { font-size:12px;display:flex;align-items:center;gap:4px; }
.stock-dot { width:6px;height:6px;border-radius:50%;display:inline-block;animation:pulse-dot 2s ease-in-out infinite; }
.product-actions { display:grid;grid-template-columns:1fr auto;gap:7px; }
.detail-btn { padding:8px 12px;font-size:13px;white-space:nowrap; }
.variant-badges { display:flex;flex-wrap:wrap;gap:6px;margin-top:-2px; }
.variant-pill { font-size:11px;padding:4px 8px;border-radius:999px;background:var(--surface-2);color:var(--text); border:0.5px solid var(--border); }
.variant-pill.muted { color:var(--text-muted); }
@keyframes pulse-dot { 0%,100%{opacity:1} 50%{opacity:0.4} }
@keyframes card-bounce {
  0% { transform: translateY(0) scale(1); }
  55% { transform: translateY(-6px) scale(1.02); }
  100% { transform: translateY(-4px) scale(1.012); }
}
@keyframes card-enter {
  from { opacity: 0; transform: translateY(10px) scale(0.985); }
  to { opacity: 1; transform: translateY(0) scale(1); }
}
.load-more-state { text-align:center;padding:1rem 0 0;color:var(--primary);font-size:13px; }
.load-more-state.muted { color:var(--text-muted); }
.load-more-meta {
  display:flex;
  justify-content:space-between;
  align-items:center;
  gap:10px;
  margin-top:12px;
  color:var(--text-muted);
  font-size:12.5px;
  flex-wrap:wrap;
}
.empty-state { text-align:center;padding:3rem;color:var(--text-muted); }
.store-footer { text-align:center;padding:2rem 0 0;font-size:13px;color:var(--text-muted); }
.wa-float {
  position:fixed;bottom:24px;right:24px;z-index:500;
  display:flex;align-items:center;gap:8px;
  background:linear-gradient(135deg,#25D366,#16a34a);color:#fff;
  padding:12px 18px;border-radius:999px;
  text-decoration:none;font-size:14px;font-weight:500;
  box-shadow:0 14px 30px rgba(37,211,102,0.28);
  transition:transform 0.2s,box-shadow 0.2s;
}
.wa-float:hover { transform:translateY(-2px);box-shadow:0 6px 20px rgba(37,211,102,0.45); }
@media (max-width: 800px) {
  .hero-panel { grid-template-columns: 1fr; }
  .hero-copy h1 { max-width: unset; }
  .aside-stats { grid-template-columns:repeat(3,1fr); }
  .nav-tag { display:none; }
}

@media (max-width: 640px) {
  .nav-inner { align-items:center; flex-wrap:nowrap; gap:8px; }
  .nav-actions { width:auto; margin-left:auto; justify-content:flex-end; flex-wrap:nowrap; gap:6px; }
  .nav-actions :global(.btn), .nav-actions :global(button) { flex:0 0 auto; padding-inline:8px; font-size:11.5px; }
  .store-container { padding-inline: 0.75rem; }
  .hero { padding-inline: 0.75rem; }
  .product-grid { grid-template-columns: repeat(2, minmax(0, 1fr)); gap: 10px; }
  .product-card { padding: 0.8rem; gap: 8px; border-radius: 14px; }
  .product-name { font-size: 13px; }
  .product-desc { display: -webkit-box; -webkit-line-clamp: 2; -webkit-box-orient: vertical; overflow: hidden; }
  .product-price { font-size: 13px; }
  .product-footer { flex-direction: column; align-items: flex-start; gap: 6px; }
  .variant-badges { gap: 4px; }
  .variant-pill { font-size: 10px; padding: 3px 7px; }
  .product-actions { grid-template-columns: 1fr; }
  .product-actions :global(.btn) { width: 100%; }
  .detail-btn { padding: 8px 10px; }
  .nav-inner { height: 58px; }
  .hero-copy h1 { font-size: 24px; }
  .hero-copy p { font-size: 13px; }
  .aside-value { font-size: 18px; }
  .aside-stats { grid-template-columns: repeat(3, minmax(0, 1fr)); }
}
</style>

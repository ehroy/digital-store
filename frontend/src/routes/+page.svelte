<script>
  import { onMount } from 'svelte';
  import { api } from '$lib/api.js';
  import { IDR } from '$lib/utils.js';
  import BuyModal from './BuyModal.svelte';
  import CheckoutModal from './CheckoutModal.svelte';
  import InvoiceModal from './InvoiceModal.svelte';

  let products = [];
  let loading = true;
  let search = '';
  let activeCategory = 'all';
  let categories = ['all'];

  let buyProduct = null;
  let checkoutData = null;
  let invoiceOrder = null;

  onMount(async () => {
    try {
      products = await api.getProducts();
      const cats = [...new Set(products.map(p => p.category))];
      categories = ['all', ...cats];
    } finally { loading = false; }
  });

  $: filtered = products.filter(p => {
    const matchCat = activeCategory === 'all' || p.category === activeCategory;
    const q = search.toLowerCase();
    const matchSearch = p.name.toLowerCase().includes(q) || p.description?.toLowerCase().includes(q);
    return matchCat && matchSearch;
  });
</script>

<svelte:head><title>DigiStore — Produk Digital</title></svelte:head>

<nav class="store-nav">
  <div class="nav-inner">
    <a href="/" class="nav-brand">
      <span class="nav-logo">🛍</span>
      <span>DigiStore</span>
    </a>
    <div style="display:flex;gap:8px;margin-left:auto">
      <a href="/cek-invoice" class="btn btn-sm">📋 Cek Invoice</a>
      <a href="/login" class="btn btn-sm">Admin →</a>
    </div>
  </div>
</nav>

<div class="hero">
  <h1>Produk Digital Terbaik</h1>
  <p>Template, ebook, plugin, dan source code berkualitas untuk mempercepat bisnis digital kamu.</p>
</div>

<div class="store-container">
  <div class="filter-row">
    <input class="input" placeholder="🔍  Cari produk..." bind:value={search} />
    <div class="cat-pills">
      {#each categories as cat}
        <button class="btn btn-sm {activeCategory===cat?'btn-primary':''}"
          on:click={() => activeCategory = cat}>
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
      {#each filtered as product (product.id)}
        {@const outOfStock = product.type === 'stock' && product.available_stock === 0}
        <div class="product-card {outOfStock?'out-of-stock':''}">
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
            <div class="product-price">{IDR(product.price)}</div>
            {#if product.type === 'stock'}
              <div class="product-stock" style="color:{outOfStock?'#8c2626':product.available_stock<5?'#854F0B':'var(--text-muted)'}">
                {outOfStock ? 'Stok habis' : product.available_stock < 5 ? `Sisa ${product.available_stock}` : 'Stok tersedia'}
              </div>
            {:else}
              <div class="product-stock">Tersedia</div>
            {/if}
          </div>
          <div style="display:grid;grid-template-columns:1fr auto;gap:7px;align-items:stretch">
            <button class="btn btn-primary"
              disabled={outOfStock}
              on:click={() => buyProduct = product}>
              {outOfStock ? 'Stok Habis' : 'Beli Sekarang'}
            </button>
            <a href="/product/{product.id}" class="btn" style="padding:8px 12px;font-size:13px;white-space:nowrap">Detail</a>
          </div>
        </div>
      {/each}
    </div>
  {/if}

  <!-- Footer -->
  <div class="store-footer">
    <p>Sudah beli? <a href="/cek-invoice" style="color:#0d5fa8;font-weight:500">Cek status &amp; download produk →</a></p>
  </div>
</div>

{#if buyProduct}
  <BuyModal product={buyProduct} on:close={()=>buyProduct=null}
    on:checkout={(e)=>{ checkoutData=e.detail; buyProduct=null; }} />
{/if}
{#if checkoutData}
  <CheckoutModal data={checkoutData} on:close={()=>checkoutData=null}
    on:done={(e)=>{ checkoutData=null; invoiceOrder=e.detail; }} />
{/if}
{#if invoiceOrder}
  <InvoiceModal order={invoiceOrder} on:close={()=>invoiceOrder=null} />
{/if}

<style>
.store-nav { position:sticky;top:0;z-index:100;background:#fff;border-bottom:0.5px solid var(--border);padding:0 1.5rem; }
.nav-inner { max-width:1100px;margin:0 auto;height:54px;display:flex;align-items:center;gap:10px; }
.nav-brand { display:flex;align-items:center;gap:8px;font-weight:500;font-size:15.5px; }
.nav-logo { background:#0d5fa8;border-radius:8px;width:28px;height:28px;display:flex;align-items:center;justify-content:center;font-size:14px; }
.hero { text-align:center;padding:2.5rem 1rem 1.5rem;background:linear-gradient(180deg,#f0f6fd 0%,var(--bg) 100%); }
.hero h1 { font-size:28px;font-weight:500;letter-spacing:-0.4px; }
.hero p { color:var(--text-muted);font-size:14.5px;margin-top:8px; }
.store-container { max-width:1100px;margin:0 auto;padding:1.25rem 1rem 3rem; }
.filter-row { display:flex;gap:10px;margin-bottom:1.5rem;flex-wrap:wrap; }
.filter-row .input { flex:1;min-width:200px; }
.cat-pills { display:flex;gap:6px;flex-wrap:wrap; }
.product-grid { display:grid;grid-template-columns:repeat(auto-fill,minmax(245px,1fr));gap:14px; }
.product-card { background:#fff;border:0.5px solid var(--border);border-radius:var(--radius-lg);padding:1.15rem;display:flex;flex-direction:column;gap:12px;transition:box-shadow 0.2s; }
.product-card:hover:not(.out-of-stock) { box-shadow:0 4px 18px rgba(0,0,0,0.08); }
.out-of-stock { opacity:0.7; }
.product-icon { font-size:40px;text-align:center;background:#f8f8f6;border-radius:var(--radius);padding:1rem; }
.product-img-wrap { border-radius:var(--radius);overflow:hidden;background:#f8f8f6;aspect-ratio:16/16; }
.product-img { width:100%;height:100%;object-fit:cover;display:block;transition:transform 0.3s; }
.product-card:hover .product-img { transform:scale(1.04); }
.product-body { flex:1; }
.product-name { font-weight:500;font-size:14.5px;line-height:1.4;margin-bottom:5px; }
.product-desc { font-size:12.5px;color:var(--text-muted);line-height:1.6; }
.product-footer { display:flex;justify-content:space-between;align-items:center;padding-top:8px;border-top:0.5px solid var(--border); }
.product-price { font-weight:500;color:#0d5fa8;font-size:16px; }
.product-stock { font-size:11.5px;color:var(--text-muted); }
.empty-state { text-align:center;padding:3rem;color:var(--text-muted); }
.store-footer { text-align:center;padding:2rem 0 0;font-size:13px;color:var(--text-muted); }
</style>

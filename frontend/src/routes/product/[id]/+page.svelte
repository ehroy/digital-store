<script>
  import { onMount } from 'svelte';
  import { page } from '$app/stores';
  import { api } from '$lib/api.js';
  import { IDR } from '$lib/utils.js';
  import BuyModal from '../../BuyModal.svelte';
  import CheckoutModal from '../../CheckoutModal.svelte';
  import InvoiceModal from '../../InvoiceModal.svelte';

  const productId = $page.params.id;
  let product = null;
  let loading = true;
  let error = '';

  let buyProduct = null;
  let checkoutData = null;
  let invoiceOrder = null;

  onMount(async () => {
    try {
      product = await api.getProduct(productId);
    } catch(e) {
      error = 'Produk tidak ditemukan.';
    } finally {
      loading = false;
    }
  });

  $: outOfStock = product?.type === 'stock' && product?.available_stock === 0;
  $: hasVariants = product?.type === 'provider' && Array.isArray(product?.variants) && product.variants.length > 0;
  $: compactVariants = hasVariants && product.variants.length > 2;
  $: providerOutOfStock = hasVariants && product.variants.every(v => v.stock_status === 'out_of_stock');
  $: stockLabel = product?.type === 'stock'
    ? (outOfStock ? 'Habis' : product.available_stock < 5 ? `Sisa ${product.available_stock}` : 'Tersedia')
    : (providerOutOfStock ? 'Habis' : hasVariants ? `${product.variants.filter(v => v.stock_status !== 'out_of_stock').length} varian aktif` : 'Tersedia');
  $: stockColor = product?.type === 'stock'
    ? (outOfStock ? '#8c2626' : product.available_stock < 5 ? '#854F0B' : '#2f5e0f')
    : (providerOutOfStock ? '#8c2626' : '#2f5e0f');

  function compactVariantName(name) {
    return (name || 'Varian').split(/\s+/).slice(0, 2).join(' ');
  }
</script>

<svelte:head>
  <title>{product?.name || 'Produk'} — Digital Murah</title>
  {#if product?.description}<meta name="description" content={product.description} />{/if}
</svelte:head>

<!-- Nav -->
<nav style="background:#fff;border-bottom:0.5px solid var(--border);padding:0 1.5rem;position:sticky;top:0;z-index:100">
  <div style="max-width:900px;margin:0 auto;height:54px;display:flex;align-items:center;gap:10px">
    <a href="/" style="display:flex;align-items:center;gap:8px;font-weight:500;font-size:15px">
      <span style="background:#0d5fa8;border-radius:8px;width:28px;height:28px;display:flex;align-items:center;justify-content:center;font-size:14px">🛍</span>
      Digital Murah
    </a>
    <span style="color:var(--text-muted);font-size:13px;margin-left:4px">/ {loading ? '…' : (product?.name || 'Produk')}</span>
    <div style="display:flex;gap:8px;margin-left:auto">
      <a href="/cek-invoice" class="btn btn-sm">📋 Cek Invoice</a>
    </div>
  </div>
</nav>

<div style="max-width:900px;margin:0 auto;padding:2rem 1rem 4rem">
  {#if loading}
    <div style="text-align:center;padding:4rem;color:var(--text-muted)">Memuat produk…</div>

  {:else if error || !product}
    <div style="text-align:center;padding:4rem">
      <div style="font-size:32px;margin-bottom:12px">😕</div>
      <div style="font-weight:500;margin-bottom:8px">Produk tidak ditemukan</div>
      <a href="/" class="btn btn-primary">Kembali ke Toko</a>
    </div>

  {:else}
    <!-- Breadcrumb -->
    <div style="font-size:12.5px;color:var(--text-muted);margin-bottom:1.5rem;display:flex;align-items:center;gap:5px">
      <a href="/" style="color:var(--text-muted)">Toko</a>
      <span>›</span>
      <span>{product.category}</span>
      <span>›</span>
      <span style="color:var(--text)">{product.name}</span>
    </div>

    <div class="product-layout">
      <!-- Kiri: Gambar -->
      <div class="product-image-col">
        {#if product.image_url}
          <div class="product-main-img-wrap">
            <img src={product.image_url} alt={product.name} class="product-main-img" />
          </div>
        {:else}
          <div class="product-main-icon">{product.icon}</div>
        {/if}
      </div>

      <!-- Kanan: Info -->
      <div class="product-info-col">
        <div style="margin-bottom:10px;display:flex;gap:7px;flex-wrap:wrap">
          <span class="badge badge-stock">{product.category}</span>
          {#if product.type === 'script'}
            <span class="badge" style="background:#EEEDFE;color:#534AB7">Layanan Jasa</span>
          {/if}
          {#if !product.active}
            <span class="badge badge-inactive">Tidak Tersedia</span>
          {/if}
        </div>

        <h1 style="font-size:22px;font-weight:500;line-height:1.35;margin-bottom:10px;letter-spacing:-0.3px">{product.name}</h1>

        <!-- Stok -->
        {#if product.type === 'stock'}
          <div style="font-size:12.5px;margin-bottom:14px">
            {#if outOfStock}
              <span style="color:{stockColor};font-weight:600">✗ Stok habis</span>
            {:else if product.available_stock < 5}
              <span style="color:{stockColor};font-weight:600">⚠️ Sisa {product.available_stock} stok</span>
            {:else}
              <span style="color:{stockColor};font-weight:600">✓ Stok tersedia</span>
            {/if}
          </div>
        {:else}
          <div style="font-size:12.5px;color:{stockColor};font-weight:600;margin-bottom:14px">
            {stockLabel}
          </div>
        {/if}

        <!-- Harga -->
        <div style="font-size:30px;font-weight:500;color:#0d5fa8;margin-bottom:20px;letter-spacing:-0.5px">
          {#if hasVariants}
            Mulai {IDR(product.price)}
          {:else}
            {IDR(product.price)}
          {/if}
        </div>

        {#if hasVariants}
          <div class="variant-preview">
            <div style="font-size:12px;color:var(--text-muted);margin-bottom:8px">Pilihan varian tersedia</div>
            {#each product.variants as variant, index}
              <div class="variant-row">
                <div>
                  <div class:compact-variant-name={compactVariants} style="font-weight:600;color:{variant.stock_status === 'out_of_stock' ? '#8c2626' : '#0f172a'}" title={variant.variant_name || 'Varian'}>
                    {compactVariants ? `${compactVariantName(variant.variant_name)} ${index + 1}` : (variant.variant_name || 'Varian')}
                  </div>
                  {#if !compactVariants}
                    <div style="font-size:11.5px;color:{variant.stock_status === 'out_of_stock' ? '#8c2626' : 'var(--text-muted)'}">
                      {#if variant.duration_label}{variant.duration_label} {/if}
                      {#if variant.account_type}{variant.account_type} {/if}
                      {#if variant.region}{variant.region} {/if}
                    </div>
                  {/if}
                </div>
                <div style="text-align:right">
                  <div style="font-weight:500;color:#0d5fa8">{IDR(variant.price)}</div>
                  <div class:compact-variant-meta={compactVariants} style="font-size:11px;font-weight:600;color:{variant.stock_status === 'out_of_stock' ? '#8c2626' : '#2f5e0f'}">
                    {variant.stock_status === 'out_of_stock' ? 'Habis' : `Stok ${variant.available_stock}`}
                  </div>
                </div>
              </div>
            {/each}
          </div>
        {/if}

        {#if hasVariants}
          <div style="font-size:12px;color:var(--text-muted);margin:8px 0 18px">
            Klik beli untuk memilih varian yang ingin diorder.
          </div>
        {/if}

        <!-- Tombol beli -->
        <button class="btn btn-primary"
          style="width:100%;padding:13px;font-size:15px;border-radius:var(--radius-lg);margin-bottom:10px"
          disabled={outOfStock || providerOutOfStock || !product.active}
          on:click={() => buyProduct = product}>
          {outOfStock || providerOutOfStock ? 'Stok Habis' : !product.active ? 'Tidak Tersedia' : 'Beli Sekarang'}
        </button>
        <a href="/" class="btn" style="width:100%;padding:10px;font-size:14px;text-align:center;display:block">
          ← Lihat Produk Lain
        </a>

        <!-- Info pengiriman -->
        <div class="delivery-info">
          {#if product.type === 'stock'}
            <div class="dinfo-row">⚡ <span>Pengiriman instan setelah pembayaran dikonfirmasi</span></div>
            <div class="dinfo-row">📧 <span>Dikirim via email beserta invoice</span></div>
          {:else}
            <div class="dinfo-row">👷 <span>Varian dipilih saat checkout sesuai stok dan harga aktif</span></div>
            <div class="dinfo-row">📧 <span>Konfirmasi order dikirim via email</span></div>
          {/if}
        </div>
      </div>
    </div>

    <!-- Deskripsi Lengkap -->
    <div class="desc-section">
      <div class="desc-header">
        <div>
          <div class="desc-kicker">Informasi Produk</div>
          <h2>Deskripsi Produk</h2>
        </div>
        <div class="desc-note">Baca detail sebelum checkout</div>
      </div>

      <div class="desc-card">
        <div class="desc-accent">✦</div>
        <div class="desc-body">
          {#each product.description.split('\n') as para}
            {#if para.trim()}
              <p>{para}</p>
            {/if}
          {/each}
        </div>
      </div>
    </div>

    <!-- Produk terkait (kategori sama) -->
  {/if}
</div>

<!-- Modals -->
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
.product-layout {
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: 2.5rem;
  margin-bottom: 2.5rem;
}
@media (max-width: 640px) { .product-layout { grid-template-columns: 1fr; gap: 1.5rem; } }

.product-image-col {}
.product-main-img-wrap {
  border-radius: var(--radius-lg);
  overflow: hidden;
  background: #f8f8f6;
  border: 0.5px solid var(--border);
  aspect-ratio: 4/3;
}
.product-main-img { width:100%;height:100%;object-fit:cover;display:block; }
.product-main-icon {
  background: #f8f8f6;
  border: 0.5px solid var(--border);
  border-radius: var(--radius-lg);
  aspect-ratio: 4/3;
  display: flex;
  align-items: center;
  justify-content: center;
  font-size: 80px;
}

.product-info-col {}

.variant-preview {
  border: 0.5px solid var(--border);
  background: #f8f8f6;
  border-radius: var(--radius);
  padding: 12px 14px;
  margin-bottom: 14px;
}
.variant-row {
  display: flex;
  justify-content: space-between;
  gap: 12px;
  padding: 9px 0;
  border-top: 0.5px solid var(--border);
}
.compact-variant-name {
  max-width: 170px;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}
.compact-variant-meta {
  white-space: nowrap;
}
.variant-row:first-of-type { border-top: 0; padding-top: 0; }

.delivery-info {
  margin-top: 16px;
  padding: 12px 14px;
  background: #f8f8f6;
  border-radius: var(--radius);
  display: flex;
  flex-direction: column;
  gap: 7px;
}
.dinfo-row {
  display: flex;
  align-items: center;
  gap: 8px;
  font-size: 13px;
  color: var(--text-muted);
}

.desc-section {
  border-top: 0.5px solid var(--border);
  padding-top: 1.75rem;
}
.desc-header {
  display: flex;
  align-items: flex-end;
  justify-content: space-between;
  gap: 12px;
  margin-bottom: 14px;
}
.desc-kicker {
  font-size: 11px;
  text-transform: uppercase;
  letter-spacing: 0.14em;
  color: var(--primary);
  font-weight: 700;
  margin-bottom: 6px;
}
.desc-header h2 {
  font-size: 17px;
  font-weight: 600;
  letter-spacing: -0.02em;
}
.desc-note {
  font-size: 12px;
  color: var(--text-muted);
  background: #f8f8f6;
  border: 0.5px solid var(--border);
  border-radius: 999px;
  padding: 7px 10px;
  white-space: nowrap;
}
.desc-card {
  display: grid;
  grid-template-columns: auto 1fr;
  gap: 14px;
  padding: 1rem 1rem 0.95rem;
  border: 0.5px solid rgba(21,93,252,0.12);
  border-radius: 18px;
  background: linear-gradient(180deg, rgba(255,255,255,0.95), rgba(248,250,252,0.95));
  box-shadow: 0 12px 28px rgba(15,23,42,0.05);
}
.desc-accent {
  width: 36px;
  height: 36px;
  border-radius: 12px;
  display: flex;
  align-items: center;
  justify-content: center;
  background: linear-gradient(135deg, rgba(21,93,252,0.12), rgba(37,211,102,0.10));
  color: var(--primary);
  font-size: 16px;
}
.desc-body {
  font-size: 14.5px;
  line-height: 1.8;
  color: var(--text);
  max-width: 720px;
}
.desc-body p { margin-bottom: 10px; }
@media (max-width: 640px) {
  .desc-header { align-items: flex-start; flex-direction: column; }
  .desc-note { white-space: normal; }
  .desc-card { grid-template-columns: 1fr; }
}
</style>

<script>
  import { createEventDispatcher } from 'svelte';
  import { IDR } from '$lib/utils.js';

  export let product;
  const dispatch = createEventDispatcher();

  let name = '';
  let email = '';
  let qty = 1;
  let error = '';
  let selectedVariantId = '';

  function prettyVariant(v) {
    const parts = [];
    if (v.account_type) parts.push(v.account_type.charAt(0).toUpperCase() + v.account_type.slice(1));
    if (v.duration_label) parts.push(v.duration_label);
    if (v.region) parts.push(v.region);
    return parts.length > 0 ? parts.join(' · ') : 'Varian';
  }

  $: isVariantProduct = product?.type === 'provider' && Array.isArray(product?.variants) && product.variants.length > 0;
  $: if (isVariantProduct && !selectedVariantId) {
    const firstAvailable = product.variants.find((v) => v.stock_status !== 'out_of_stock') || product.variants[0];
    selectedVariantId = firstAvailable?.product_id ? String(firstAvailable.product_id) : '';
  }
  $: selectedVariant = isVariantProduct
    ? product.variants.find((v) => String(v.product_id) === selectedVariantId)
    : null;
  $: activeStock = isVariantProduct
    ? selectedVariant
    : (product?.type === 'stock' ? { stock_status: product.available_stock > 0 ? 'available' : 'out_of_stock', available_stock: product.available_stock } : null);
  $: unitPrice = isVariantProduct ? (selectedVariant?.price || 0) : (product?.price || 0);
  $: displayName = isVariantProduct && selectedVariant
    ? `${product.name} · ${selectedVariant.variant_name}`
    : product?.name;

 function submit() {
  if (!name.trim()) {
    error = 'Nama wajib diisi.';
    return;
  }

  if (!email.includes('@') || !email.includes('.')) {
    error = 'Format email tidak valid.';
    return;
  }

  if (isVariantProduct) {
    if (!selectedVariant) {
      error = 'Pilih varian terlebih dahulu.';
      return;
    }
    if (selectedVariant.stock_status === 'out_of_stock') {
      error = 'Varian ini sedang habis.';
      return;
    }
    if (selectedVariant.stock_status === 'available' && selectedVariant.available_stock > 0 && qty > selectedVariant.available_stock) {
      error = `Jumlah maksimal untuk varian ini adalah ${selectedVariant.available_stock}.`;
      return;
    }

    error = '';
    dispatch('checkout', {
      product: { ...product, id: selectedVariant.product_id, name: displayName, price: unitPrice },
      variant: selectedVariant,
      name,
      email,
      qty,
      total: unitPrice * qty
    });
    return;
  }

  // ❌ NON-PROVIDER (pakai stock biasa)
  if (product.type === 'stock' && (qty < 1 || qty > product.available_stock)) {
    error = `Jumlah harus antara 1 dan ${product.available_stock}.`;
    return;
  }

  error = '';
  dispatch('checkout', {
    product,
    name,
    email,
    qty,
    total: unitPrice * qty
  });
}
  function overlay(e) { if (e.target === e.currentTarget) dispatch('close'); }
</script>

<div class="modal-overlay" on:click={overlay} role="dialog" aria-modal="true">
  <div class="modal-box" style="max-width:460px">
    <div class="modal-header">
      <span class="modal-title">Beli Produk</span>
      <button class="modal-close" on:click={() => dispatch('close')}>×</button>
    </div>

    <!-- Product summary -->
    <div class="product-summary">
      <span class="picon">{product.icon}</span>
      <div>
        <div style="font-weight:500;font-size:14px">{product.name}</div>
        <div style="color:#0d5fa8;font-weight:500;font-size:15px">{IDR(unitPrice || product.price)} / pcs</div>
      </div>
    </div>

    {#if isVariantProduct}
      <div>
        <label class="field-label">Pilih Varian</label>
        <div class="variant-list">
          {#each product.variants as variant}
            <button
              class="variant-item {String(variant.product_id)===selectedVariantId?'active':''}"
              on:click={() => selectedVariantId = String(variant.product_id)}
              type="button"
            >
              <div>
                <div style="font-weight:500">{variant.variant_name || 'Varian'}</div>
                <div style="font-size:11.5px;color:var(--text-muted)">{prettyVariant(variant)}</div>
              </div>
              <div style="text-align:right">
                <div style="font-weight:500;color:#0d5fa8">{IDR(variant.price)}</div>
                <div style="font-size:11px;color:var(--text-muted)">
                  {variant.stock_status === 'out_of_stock' ? 'Habis' : variant.stock_status === 'manual' ? 'Manual' : `Stok asli ${variant.available_stock}`}
                </div>
              </div>
            </button>
          {/each}
        </div>
      </div>
    {/if}

    <div class="form-grid">
      <div>
        <label class="field-label">Nama Lengkap</label>
        <input class="input" placeholder="Masukkan nama lengkap" bind:value={name} />
      </div>
      <div>
        <label class="field-label">Email (untuk invoice)</label>
        <input class="input" type="email" placeholder="email@example.com" bind:value={email} />
      </div>
      <div>
        <label class="field-label">Jumlah</label>
       <div class="qty-row">
         <button class="btn btn-sm" type="button" on:click={() => qty = Math.max(1, qty - 1)}>−</button>

        <span class="qty-val">{qty}</span>

        <button
          type="button"
          class="btn btn-sm"
          on:click={() => {
            if (isVariantProduct) {
              if (selectedVariant?.stock_status === 'available' && selectedVariant.available_stock > 0) {
                qty = Math.min(selectedVariant.available_stock, qty + 1);
              } else {
                qty = qty + 1;
              }
            } else if (product.type === 'provider' && product.provider_status === 'available') {
              qty = qty + 1;
            } else {
              qty = Math.min(product.available_stock, qty + 1);
            }
          }}
        >
          +
        </button>

        <span style="font-size:12px;color:var(--text-muted)">
          {#if isVariantProduct}
            {#if selectedVariant?.stock_status === 'available' && selectedVariant.available_stock > 0}
              Stok asli {selectedVariant.available_stock}
            {:else if selectedVariant?.stock_status === 'manual'}
              Stock manual
            {:else}
              Varian tidak tersedia
            {/if}
          {:else if product.type === 'provider' && product.provider_status === 'available'}
            Stock Tergantung penyedia
          {:else}
            Maks {product.available_stock}
          {/if}
        </span>
      </div>
      </div>

      {#if error}<div class="alert-error">{error}</div>{/if}

      <div class="total-row">
        <div>
          <div style="font-size:12px;color:var(--text-muted)">Total Pembayaran</div>
          <div style="font-weight:500;font-size:19px">{IDR((unitPrice || product.price) * qty)}</div>
        </div>
        <button class="btn btn-primary" type="button" on:click={submit}>Lanjut ke Pembayaran →</button>
      </div>
    </div>
  </div>
</div>

<style>
.product-summary {
  display: flex; align-items: center; gap: 12px;
  background: #f8f8f6; border-radius: var(--radius);
  padding: 12px 14px; margin-bottom: 16px;
}
.picon { font-size: 28px; }
.form-grid { display: flex; flex-direction: column; gap: 14px; }
.qty-row { display: flex; align-items: center; gap: 12px; }
.qty-val { font-weight: 500; font-size: 17px; min-width: 28px; text-align: center; }
.variant-list { display: flex; flex-direction: column; gap: 8px; margin-top: 8px; }
.variant-item {
  display: flex; justify-content: space-between; align-items: center; gap: 12px;
  width: 100%; text-align: left;
  border: 0.5px solid var(--border);
  border-radius: var(--radius);
  padding: 10px 12px;
  background: #fff;
  cursor: pointer;
}
.variant-item.active { border: 1.5px solid #0d5fa8; background: #fafeff; }
.total-row {
  display: flex; justify-content: space-between; align-items: center;
  border-top: 0.5px solid var(--border); padding-top: 14px;
}
</style>

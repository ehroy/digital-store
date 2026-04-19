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
    const firstAvailable = product.variants.find((v) => v.stock_status !== 'out_of_stock');
    selectedVariantId = firstAvailable?.product_id ? String(firstAvailable.product_id) : '';
  }
  $: selectedVariant = isVariantProduct
    ? product.variants.find((v) => String(v.product_id) === selectedVariantId)
    : null;
  $: selectedVariantAvailableStock = selectedVariant
    ? Number(selectedVariant.available_stock) || Number(selectedVariant.internal_stock) || 0
    : 0;
  $: unitPrice = isVariantProduct ? (selectedVariant?.price || 0) : (product?.price || 0);
  $: displayName = isVariantProduct && selectedVariant
    ? `${product.name} · ${selectedVariant.variant_name}`
    : product?.name;
  $: hasAvailableVariant = isVariantProduct && product.variants.some((v) => v.stock_status !== 'out_of_stock');

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
        error = hasAvailableVariant ? 'Pilih varian terlebih dahulu.' : 'Semua varian sedang habis.';
        return;
      }
    if (selectedVariant.stock_status === 'out_of_stock') {
      error = 'Varian ini sedang habis.';
      return;
    }
    if (selectedVariantAvailableStock > 0 && qty > selectedVariantAvailableStock) {
      error = `Jumlah maksimal untuk varian ini adalah ${selectedVariantAvailableStock}.`;
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

  if (product.type === 'provider' && Number(product.available_stock) > 0 && qty > Number(product.available_stock)) {
    error = `Jumlah maksimal untuk produk ini adalah ${product.available_stock}.`;
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

<div class="modal-overlay" on:click={overlay} on:keydown={(e)=>e.key==='Escape'&&dispatch('close')} role="dialog" aria-modal="true" tabindex="-1">
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
        <div style="color:var(--primary);font-weight:500;font-size:15px">{IDR(unitPrice || product.price)} / pcs</div>
      </div>
    </div>

    {#if isVariantProduct}
      <div>
        <div class="field-label">Pilih Varian</div>
        <div class="variant-list">
          {#each product.variants as variant}
            <button
              class="variant-item {String(variant.product_id)===selectedVariantId?'active':''} {variant.stock_status === 'out_of_stock' ? 'sold-out' : ''}"
              on:click={() => variant.stock_status !== 'out_of_stock' && (selectedVariantId = String(variant.product_id))}
              type="button"
              disabled={variant.stock_status === 'out_of_stock'}
            >
              <div>
                <div style="font-weight:500">{variant.variant_name || 'Varian'}</div>
                <div style="font-size:11.5px;color:var(--text-muted)">{prettyVariant(variant)}</div>
              </div>
              <div style="text-align:right">
                <div style="font-weight:500;color:var(--primary)">{IDR(variant.price)}</div>
                <div class="variant-stock {variant.stock_status === 'out_of_stock' ? 'status-out' : 'status-available'}">
                  {variant.stock_status === 'out_of_stock' ? 'Habis' : `Stok ${variant.available_stock}`}
                </div>
              </div>
            </button>
          {/each}
        </div>
      </div>
    {/if}

    <div class="form-grid">
      <div>
        <div class="field-label">Nama Lengkap</div>
        <input class="input" placeholder="Masukkan nama lengkap" bind:value={name} />
      </div>
      <div>
        <div class="field-label">Email (untuk invoice)</div>
        <input class="input" type="email" placeholder="email@example.com" bind:value={email} />
      </div>
      <div>
        <div class="field-label">Jumlah</div>
       <div class="qty-row">
         <button class="btn btn-sm" type="button" on:click={() => qty = Math.max(1, qty - 1)}>−</button>

        <span class="qty-val">{qty}</span>

        <button
          type="button"
          class="btn btn-sm"
          disabled={isVariantProduct ? !selectedVariant || selectedVariant.stock_status === 'out_of_stock' : product.type === 'stock' ? product.available_stock <= 0 : false}
          on:click={() => {
            if (isVariantProduct) {
              if (selectedVariant?.stock_status === 'available' && selectedVariant.available_stock > 0) {
                qty = Math.min(selectedVariant.available_stock, qty + 1);
              } else {
                qty = qty + 1;
              }
            } else if (product.type === 'provider' && Number(product.available_stock) > 0) {
              qty = Math.min(Number(product.available_stock), qty + 1);
            } else {
              qty = Math.min(product.available_stock, qty + 1);
            }
          }}
        >
          +
        </button>

          <span class="qty-hint">
          {#if isVariantProduct}
            {#if selectedVariantAvailableStock > 0}
              Stok {selectedVariantAvailableStock}
            {:else if selectedVariant?.stock_status === 'manual'}
              Stock manual
            {:else}
              Varian tidak tersedia
            {/if}
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
        <button class="btn btn-primary" type="button" on:click={submit} disabled={isVariantProduct && !hasAvailableVariant}>Lanjut ke Pembayaran →</button>
      </div>
    </div>
  </div>
</div>

<style>
.product-summary {
  display: flex; align-items: center; gap: 12px;
  background: var(--surface-2); border-radius: var(--radius);
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
  background: var(--surface);
  cursor: pointer;
}
.variant-item.active { border: 1.5px solid var(--primary); background: var(--primary-bg); }
.variant-item.sold-out {
  cursor: not-allowed;
  opacity: 0.62;
  background: var(--surface-2);
}
.variant-item:disabled {
  cursor: not-allowed;
}
.variant-stock {
  font-size: 11px;
  font-weight: 600;
}
.variant-stock.status-available { color: #2f5e0f; }
.variant-stock.status-out { color: #8c2626; }
.qty-hint { font-size:12px;color:var(--text-muted); }
.total-row {
  display: flex; justify-content: space-between; align-items: center;
  border-top: 0.5px solid var(--border); padding-top: 14px;
}
</style>

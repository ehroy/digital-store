<script>
  import { createEventDispatcher } from 'svelte';
  import { IDR } from '$lib/utils.js';

  export let product;
  console.log(product)
  const dispatch = createEventDispatcher();

  let name = '';
  let email = '';
  let qty = 1;
  let error = '';

  function submit() {
    if (!name.trim()) { error = 'Nama wajib diisi.'; return; }
    if (!email.includes('@') || !email.includes('.')) { error = 'Format email tidak valid.'; return; }
    if (qty < 1 || qty > product.available_stock) { error = `Jumlah harus antara 1 dan ${product.available_stock}.`; return; }
    error = '';
    dispatch('checkout', { product, name, email, qty, total: product.price * qty });
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
        <div style="color:#0d5fa8;font-weight:500;font-size:15px">{IDR(product.price)} / pcs</div>
      </div>
    </div>

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
          <button class="btn btn-sm" on:click={() => qty = Math.max(1, qty - 1)}>−</button>
          <span class="qty-val">{qty}</span>
          <button class="btn btn-sm" on:click={() => qty = Math.min(product.available_stock, qty + 1)}>+</button>
          <span style="font-size:12px;color:var(--text-muted)">Maks {product.available_stock}</span>
        </div>
      </div>

      {#if error}<div class="alert-error">{error}</div>{/if}

      <div class="total-row">
        <div>
          <div style="font-size:12px;color:var(--text-muted)">Total Pembayaran</div>
          <div style="font-weight:500;font-size:19px">{IDR(product.price * qty)}</div>
        </div>
        <button class="btn btn-primary" on:click={submit}>Lanjut ke Pembayaran →</button>
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
.total-row {
  display: flex; justify-content: space-between; align-items: center;
  border-top: 0.5px solid var(--border); padding-top: 14px;
}
</style>

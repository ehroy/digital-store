<script>
  import { createEventDispatcher, onMount } from 'svelte';
  import { api } from '$lib/api.js';
  import { IDR } from '$lib/utils.js';
  import { goto } from '$app/navigation';

  export let data; // { product, name, email, qty, total }
  const dispatch = createEventDispatcher();

  let loading = false;
  let loadingMethods = true;
  let error = '';
  let gatewayActive = false;
  let gatewayProvider = 'manual';

  function prettyVariant(v) {
    const parts = [];
    if (v?.account_type) parts.push(v.account_type.charAt(0).toUpperCase() + v.account_type.slice(1));
    if (v?.duration_label) parts.push(v.duration_label);
    if (v?.region) parts.push(v.region);
    return parts.length > 0 ? parts.join(' · ') : 'Varian';
  }

  onMount(async () => {
    try {
      const res = await api.getPaymentMethods();
      gatewayActive = (res.methods || []).some((m) => m.id === 'qris');
      gatewayProvider = res.provider || 'manual';
    } catch(e) {
      error = 'Gagal memuat metode pembayaran: ' + e.message;
    } finally {
      loadingMethods = false;
    }
  });

  async function confirm() {
    if (!gatewayActive) return;
    loading = true; error = '';
    try {
      const res = await api.placeOrder({
        product_id: data.product.id,
        buyer_name: data.name,
        buyer_email: data.email,
        qty: data.qty,
        pay_method: 'qris',
      });

      // Generate view token → simpan di sessionStorage (akses invoice tanpa email)
      let viewToken = '';
      try {
        const tk = await api.generateInvoiceToken(res.invoice_no, data.email);
        viewToken = tk.token || '';
      } catch {}

      if (typeof sessionStorage !== 'undefined') {
        sessionStorage.setItem('inv_token_' + res.invoice_no, viewToken || data.email);
      }

      // Jika gateway aktif dan ada payment URL → redirect ke portal pembayaran
      const paymentUrl = res.pay_url || res.redirect_url;

      if (paymentUrl) {
        const cred = encodeURIComponent(viewToken || data.email);
        goto(`/checkout/portal?invoice=${res.invoice_no}&url=${encodeURIComponent(paymentUrl)}&cred=${cred}`);
      } else {
        goto(`/payment/${res.invoice_no}`);
      }
    } catch(e) {
      error = e.message;
      loading = false;
    }
  }

  function overlay(e) { if (e.target === e.currentTarget) dispatch('close'); }
</script>

<div class="modal-overlay" on:click={overlay} on:keydown={(e)=>e.key==='Escape'&&dispatch('close')} role="dialog" aria-modal="true" tabindex="-1">
  <div class="modal-box" style="max-width:500px">
    <div class="modal-header">
      <span class="modal-title">Konfirmasi &amp; Pembayaran</span>
      <button class="modal-close" on:click={() => dispatch('close')}>×</button>
    </div>

    <!-- Order summary -->
    <div class="order-summary">
      <div>
        <div style="font-size:12px;color:var(--text-muted);margin-bottom:3px">
          {data.product.name}{#if data.variant?.variant_name} · {data.variant.variant_name}{/if} × {data.qty}
        </div>
        {#if data.variant}
          <div style="font-size:11.5px;color:var(--text-muted);margin-bottom:3px">{prettyVariant(data.variant)}</div>
        {/if}
        <div style="font-weight:500;font-size:20px;color:var(--primary)">{IDR(data.total)}</div>
      </div>
      <span style="font-size:30px">{data.product.icon}</span>
    </div>

    {#if loadingMethods}
      <div style="padding:1.5rem;text-align:center;color:var(--text-muted);font-size:13px">
        <div style="font-size:20px;margin-bottom:6px">🔄</div>Memuat metode pembayaran…
      </div>

    {:else if !gatewayActive}
      <div class="alert-error">QRIS belum tersedia. Hubungi admin.</div>

    {:else}
      <div class="qris-card">
        <div style="font-size:16px">📱 QRIS</div>
        <div style="font-size:12.5px;color:var(--text-muted);margin-top:4px">
          Scan QR untuk pembayaran. Produk dikirim otomatis setelah pembayaran terverifikasi.
        </div>
      </div>

      {#if gatewayProvider === 'dompetx'}
        <div class="info-box" style="margin-bottom:10px;background:var(--warning-bg);color:var(--warning-fg)">
          DompetX dapat menambahkan fee layanan sesuai ketentuan gateway. Total akhir akan mengikuti nominal dari portal pembayaran.
        </div>
      {/if}

      <div class="info-box" style="margin-bottom:10px">
        ⚡ Pembayaran hanya tersedia via QRIS.
      </div>
    {/if}

    {#if error}<div class="alert-error" style="margin-top:8px">{error}</div>{/if}

    <button class="btn btn-primary"
      style="width:100%;padding:11px;margin-top:12px;font-size:14px"
      on:click={confirm} disabled={loading || loadingMethods || !gatewayActive}>
      {loading ? 'Memproses…' : gatewayActive ? 'Lanjut ke Halaman Bayar →' : 'QRIS belum aktif'}
    </button>
  </div>
</div>

<style>
.order-summary { display:flex;justify-content:space-between;align-items:center;background:var(--surface-2);border-radius:var(--radius);padding:12px 16px;margin-bottom:18px; }
.qris-card { border:0.5px solid var(--border);border-radius:var(--radius);padding:12px 14px;background:var(--surface);margin-bottom:12px; }
.info-box { background:var(--info-bg);border-radius:var(--radius);padding:9px 13px;font-size:12.5px;color:var(--info-fg); }
</style>

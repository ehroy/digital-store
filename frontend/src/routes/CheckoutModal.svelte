<script>
  import { createEventDispatcher, onMount } from 'svelte';
  import { api } from '$lib/api.js';
  import { IDR } from '$lib/utils.js';
  import { goto } from '$app/navigation';

  export let data; // { product, name, email, qty, total }
  const dispatch = createEventDispatcher();

  let methods = [];
  let selectedMethod = null;
  let loading = false;
  let loadingMethods = true;
  let error = '';
  let gatewayActive = false;

  onMount(async () => {
    try {
      const res = await api.getPaymentMethods();
      methods = res.methods || [];
      gatewayActive = res.gateway_active || false;
      if (methods.length > 0) selectedMethod = methods[0].id;
    } catch(e) {
      error = 'Gagal memuat metode pembayaran: ' + e.message;
    } finally {
      loadingMethods = false;
    }
  });

  async function confirm() {
    if (!selectedMethod) return;
    loading = true; error = '';
    try {
      const res = await api.placeOrder({
        product_id: data.product.id,
        buyer_name: data.name,
        buyer_email: data.email,
        qty: data.qty,
        pay_method: selectedMethod,
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
      if (res.pay_url) {
        const cred = encodeURIComponent(viewToken || data.email);
        goto(`/checkout/portal?invoice=${res.invoice_no}&url=${encodeURIComponent(res.pay_url)}&cred=${cred}`);
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

<div class="modal-overlay" on:click={overlay} role="dialog" aria-modal="true">
  <div class="modal-box" style="max-width:500px">
    <div class="modal-header">
      <span class="modal-title">Konfirmasi &amp; Pembayaran</span>
      <button class="modal-close" on:click={() => dispatch('close')}>×</button>
    </div>

    <!-- Order summary -->
    <div class="order-summary">
      <div>
        <div style="font-size:12px;color:var(--text-muted);margin-bottom:3px">{data.product.name} × {data.qty}</div>
        <div style="font-weight:500;font-size:20px;color:#0d5fa8">{IDR(data.total)}</div>
      </div>
      <span style="font-size:30px">{data.product.icon}</span>
    </div>

    {#if loadingMethods}
      <div style="padding:1.5rem;text-align:center;color:var(--text-muted);font-size:13px">
        <div style="font-size:20px;margin-bottom:6px">🔄</div>Memuat metode pembayaran…
      </div>

    {:else if methods.length === 0}
      <div class="alert-error">Tidak ada metode pembayaran yang tersedia. Hubungi admin.</div>

    {:else}
      <label class="field-label" style="margin-bottom:8px">Pilih Metode Pembayaran</label>
      <div class="methods">
        {#each methods as m}
          <div class="method-option {selectedMethod===m.id?'selected':''}"
            on:click={() => selectedMethod = m.id}
            role="radio" aria-checked={selectedMethod===m.id} tabindex="0"
            on:keydown={(e) => e.key==='Enter' && (selectedMethod=m.id)}>
            <div class="radio-dot {selectedMethod===m.id?'active':''}"></div>
            <div style="flex:1;min-width:0">
              <div style="font-size:13.5px;font-weight:{selectedMethod===m.id?500:400};display:flex;align-items:center;gap:6px">
                <span>{m.icon}</span> {m.label}
                {#if m.id === 'gateway'}
                  <span style="font-size:11px;background:#E6F1FB;color:#185FA5;padding:1px 7px;border-radius:999px;font-weight:500">Otomatis</span>
                {/if}
              </div>
              <div class="mono" style="color:var(--text-muted);font-size:11.5px;margin-top:2px;overflow:hidden;text-overflow:ellipsis;white-space:nowrap;max-width:340px">{m.detail}</div>
            </div>
          </div>
        {/each}
      </div>

      {#if gatewayActive && selectedMethod === 'gateway'}
        <div class="info-box" style="margin-bottom:10px">
          ⚡ Kamu akan diarahkan ke halaman pembayaran. Pilih metode (transfer, QRIS, e-Wallet) di sana.
          Produk dikirim <strong>otomatis</strong> setelah pembayaran berhasil.
        </div>
      {:else}
        <div class="info-box" style="margin-bottom:10px">
          📧 Invoice dikirim ke <strong>{data.email}</strong> setelah pesanan dibuat.
        </div>
      {/if}
    {/if}

    {#if error}<div class="alert-error" style="margin-top:8px">{error}</div>{/if}

    <button class="btn btn-primary"
      style="width:100%;padding:11px;margin-top:12px;font-size:14px"
      on:click={confirm} disabled={loading || loadingMethods || !selectedMethod}>
      {loading ? 'Memproses…' : gatewayActive ? 'Lanjut ke Halaman Bayar →' : 'Konfirmasi Pesanan →'}
    </button>
  </div>
</div>

<style>
.order-summary { display:flex;justify-content:space-between;align-items:center;background:#f8f8f6;border-radius:var(--radius);padding:12px 16px;margin-bottom:18px; }
.methods { display:flex;flex-direction:column;gap:7px;margin-bottom:12px; }
.method-option { display:flex;align-items:center;gap:10px;padding:10px 13px;border:0.5px solid var(--border);border-radius:var(--radius);cursor:pointer;background:#fff;transition:border-color 0.12s; }
.method-option.selected { border:1.5px solid #0d5fa8;background:#fafeff; }
.radio-dot { width:15px;height:15px;border-radius:50%;border:2px solid var(--border-md);flex-shrink:0;transition:all 0.12s; }
.radio-dot.active { background:#0d5fa8;border-color:#0d5fa8; }
.info-box { background:#E6F1FB;border-radius:var(--radius);padding:9px 13px;font-size:12.5px;color:#185FA5; }
</style>

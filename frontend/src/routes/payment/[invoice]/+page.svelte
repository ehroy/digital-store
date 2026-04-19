<script>
  import { onMount, onDestroy } from 'svelte';
  import { page } from '$app/stores';
  import { api } from '$lib/api.js';
  import { IDR, fmtDateTime, PAY_LABEL } from '$lib/utils.js';
  import ThemeToggle from '$lib/ThemeToggle.svelte';
  import { toDataURL } from 'qrcode';

  const invoiceNo = $page.params.invoice;

  // Ambil credential (token atau email) dari sessionStorage
  // Token di-generate saat checkout berhasil dan disimpan dengan key inv_token_{invoiceNo}
  function getCred() {
    if (typeof sessionStorage === 'undefined') return '';
    return sessionStorage.getItem('inv_token_' + invoiceNo) || '';
  }

  let data = null;
  let loading = true;
  let error = '';
  let pollInterval = null;
  let countdown = 8;
  let countdownTimer = null;
  let pollCount = 0;
  let lastChecked = null;
  let cred = '';
  let qrisDataUrl = '';
  let showDetails = false;

  $: statusKey = data?.status === 'pending' ? 'waiting_payment' : data?.status;
  $: isFinal = data && ['paid','cancelled','expired','failed'].includes(statusKey);
  $: gatewayActive = data?.gateway_redirect_url || data?.gateway_pay_url || data?.gateway_pay_code || data?.gateway_qris_string || data?.gateway_qris_image_url;
  $: paymentLink = data?.gateway_redirect_url || data?.gateway_pay_url || '';
  $: qrisString = data?.gateway_qris_string || data?.gateway_pay_code || '';
  $: qrisImage = data?.gateway_qris_image_url || '';
  function isURL(s) { return s?.startsWith('http'); }

  async function fetchStatus() {
    try {
      data = await api.getInvoice(invoiceNo, cred);
      await syncQrisPreview(data);
      pollCount++; lastChecked = new Date(); error = '';
      if (isFinal) stopPolling();
    } catch(e) {
      // Jika 400/403 berarti tidak ada cred — arahkan ke cek-invoice dengan form email
      if (e.message.includes('verifikasi') || e.message.includes('email') || e.message.includes('token')) {
        stopPolling();
        error = 'sesi_habis';
      } else {
        error = e.message;
      }
    } finally { loading = false; }
  }

  function startPolling() {
    fetchStatus();
    pollInterval = setInterval(fetchStatus, 8000);
    countdownTimer = setInterval(() => {
      countdown = isFinal ? 0 : (countdown <= 1 ? 8 : countdown - 1);
    }, 1000);
  }
  function stopPolling() {
    clearInterval(pollInterval); clearInterval(countdownTimer);
    pollInterval = null;
  }
  function refresh() { countdown = 8; fetchStatus(); }

  async function syncQrisPreview(payload) {
    const code = payload?.gateway_qris_string || payload?.gateway_pay_code || '';
    const image = payload?.gateway_qris_image_url || '';
    if (!code) {
      qrisDataUrl = '';
      return;
    }
    if (image || typeof window === 'undefined') {
      qrisDataUrl = '';
      return;
    }
    try {
      qrisDataUrl = await toDataURL(code, { margin: 1, scale: 6 });
    } catch {
      qrisDataUrl = '';
    }
  }

  onMount(() => {
    cred = getCred();
    startPolling();
  });
  onDestroy(stopPolling);

  const SC = {
    paid:            { icon:'✅', color:'var(--success-fg)', bg:'var(--success-bg)', title:'Pembayaran Dikonfirmasi!',   desc:'Produk Anda sudah siap di bawah.' },
    waiting_payment: { icon:'⏳', color:'var(--warning-fg)', bg:'var(--warning-bg)', title:'Menunggu Pembayaran',        desc:'Halaman ini diperbarui otomatis.' },
    verifying:       { icon:'🔎', color:'var(--info-fg)', bg:'var(--info-bg)', title:'Pembayaran Sedang Diverifikasi', desc:'Sistem sedang memastikan pembayaran.' },
    script_executed: { icon:'⚙️', color:'var(--info-fg)', bg:'var(--info-bg)', title:'Pesanan Diproses',          desc:'Tim akan menghubungi dalam 1×24 jam.' },
    expired:         { icon:'⌛', color:'var(--danger-fg)', bg:'var(--danger-bg)', title:'Waktu Pembayaran Habis',    desc:'Pesanan otomatis dibatalkan.' },
    failed:          { icon:'❌', color:'var(--danger-fg)', bg:'var(--danger-bg)', title:'Pembayaran Gagal',          desc:'Silakan coba lagi.' },
    cancelled:       { icon:'✗',  color:'var(--danger-fg)', bg:'var(--danger-bg)', title:'Pesanan Dibatalkan',        desc:'Hubungi kami jika ada pertanyaan.' },
  };

  $: timeLeft = (() => {
    if (!data?.expired_at) return null;
    const diff = new Date(data.expired_at) - new Date();
    if (diff <= 0) return null;
    const h = Math.floor(diff/3600000);
    const m = Math.floor((diff%3600000)/60000);
    const s = Math.floor((diff%60000)/1000);
    return h > 0 ? `${h}j ${m}m` : m > 0 ? `${m}m ${s}s` : `${s}s`;
  })();
</script>

<svelte:head><title>Pembayaran {invoiceNo} — Digital Murah</title></svelte:head>

<nav class="invoice-nav">
  <div class="invoice-nav-inner">
    <a href="/" class="invoice-brand">
      <span style="background:linear-gradient(135deg,var(--primary),var(--primary-2));color:var(--primary-fg);border-radius:8px;width:28px;height:28px;display:flex;align-items:center;justify-content:center;font-size:14px">🛍</span>
      Digital Murah
    </a>
    <div class="invoice-actions">
      <span class="invoice-meta mono">{invoiceNo}</span>
    </div>
  </div>
</nav>

<ThemeToggle floating />

<div style="max-width:780px;margin:0 auto;padding:1.5rem 1rem 3rem">

  {#if loading}
    <div style="text-align:center;padding:4rem;color:var(--text-muted)">
      <div style="font-size:28px;margin-bottom:10px">🔄</div>Memuat…
    </div>

  {:else if error === 'sesi_habis'}
    <!-- Sesi habis / tidak ada token — arahkan ke cek invoice dengan email -->
    <div class="card" style="text-align:center;padding:2.5rem;max-width:440px;margin:2rem auto">
      <div style="font-size:36px;margin-bottom:14px">🔐</div>
      <div style="font-weight:500;font-size:16px;margin-bottom:8px">Verifikasi Diperlukan</div>
      <p style="font-size:13.5px;color:var(--text-muted);margin-bottom:20px">
        Untuk melihat invoice ini, masukkan email yang digunakan saat pembelian.
      </p>
      <a href="/cek-invoice?no={invoiceNo}"
        class="btn btn-primary" style="display:inline-block;padding:10px 24px;font-size:14px;text-decoration:none">
        Cek dengan Email →
      </a>
    </div>

  {:else if error}
    <div class="card" style="text-align:center;padding:2.5rem">
      <div style="font-size:32px;margin-bottom:10px">❌</div>
      <div style="font-weight:500;margin-bottom:16px">{error}</div>
      <a href="/cek-invoice?no={invoiceNo}" class="btn btn-primary">Cek Invoice →</a>
    </div>

  {:else if data}
    {@const sc = SC[statusKey] || SC.waiting_payment}

    <!-- Status banner -->
    <div style="background:{sc.bg};border-radius:var(--radius-lg);padding:1.25rem 1.5rem;margin-bottom:16px;display:flex;align-items:center;justify-content:space-between;gap:12px">
      <div style="display:flex;align-items:center;gap:14px">
        <span style="font-size:36px">{sc.icon}</span>
        <div>
          <div style="font-weight:500;font-size:17px;color:{sc.color}">{sc.title}</div>
          <div style="font-size:13px;color:{sc.color};opacity:0.8;margin-top:3px">{sc.desc}</div>
        </div>
      </div>
      {#if !isFinal}
        <div style="text-align:right;flex-shrink:0">
          {#if timeLeft}
            <div style="font-size:11px;color:{sc.color};opacity:0.7">Berlaku sampai</div>
            <div style="font-size:18px;font-weight:500;color:{sc.color};font-family:'JetBrains Mono',monospace">{timeLeft}</div>
          {/if}
          <div style="font-size:11px;color:{sc.color};opacity:0.6;margin-top:4px">Auto-refresh: {countdown}s</div>
          <button class="btn btn-sm" style="margin-top:4px;font-size:11px;padding:3px 10px;border-color:{sc.color}30;color:{sc.color}"
            on:click={refresh}>🔄 Refresh</button>
        </div>
      {/if}
    </div>

    <div class="pay-grid">
      <!-- Kiri: detail + instruksi -->
      <div style="display:flex;flex-direction:column;gap:12px">
        <button class="detail-toggle" type="button" on:click={() => showDetails = !showDetails}>
          <span>Detail Pesanan</span>
          <span>{showDetails ? 'Sembunyikan' : 'Lihat detail'}</span>
        </button>

        <div class:detail-open={showDetails} class="detail-panel card" style="padding:0;overflow:hidden">
          <div class="detail-head">Detail Pesanan</div>
          <table class="detail-table">
            <tbody>
              {#each [
                ['Produk', data.product_name],
                ['Pembeli', data.buyer_name],
                ['Jumlah', `${data.qty} pcs`],
                ['Metode', PAY_LABEL[data.pay_method] || data.pay_method],
              ] as [l,v], i}
                <tr style="background:{i%2===0?'var(--surface-2)':'var(--surface)'}">
                  <td>{l}</td>
                  <td>{v}</td>
                </tr>
              {/each}
              <tr class="detail-total">
                <td>Total</td>
                <td>{IDR(data.total)}</td>
              </tr>
            </tbody>
          </table>
        </div>

        {#if !isFinal}
          <div class="card" style="padding:0;overflow:hidden">
            <div style="padding:0.9rem 1.25rem;font-weight:500;font-size:14px;border-bottom:0.5px solid var(--border)">Pembayaran QRIS</div>
            <div style="padding:1rem 1.25rem;display:flex;flex-direction:column;gap:12px">
              {#if qrisImage || qrisDataUrl}
                <img src={qrisImage || qrisDataUrl} alt="QRIS" style="width:100%;max-width:280px;margin:0 auto;border-radius:12px;border:0.5px solid var(--border);background:var(--surface);padding:8px" />
              {/if}
              {#if !qrisImage && !qrisString && paymentLink}
                <div class="info-box">QRIS belum muncul dari gateway. Gunakan link berikut untuk melanjutkan pembayaran.</div>
              {/if}
              {#if paymentLink}
                <a href={paymentLink} target="_blank" rel="noopener"
                  style="display:inline-flex;justify-content:center;background:linear-gradient(135deg,var(--primary),var(--primary-2));color:var(--primary-fg);padding:12px 18px;border-radius:8px;font-size:14px;font-weight:600;text-decoration:none">
                  🔗 Buka Halaman Bayar
                </a>
              {/if}
            </div>
          </div>
        {/if}

        {#if lastChecked && !isFinal}
          <div style="font-size:11.5px;color:var(--text-hint);text-align:center">
            Dicek {pollCount}× · terakhir {fmtDateTime(lastChecked.toISOString())}
          </div>
        {/if}
      </div>

      <!-- Kanan: produk yang diterima -->
      <div>
        {#if statusKey === 'paid' && data.delivered_items?.length}
          <div class="card" style="padding:0;overflow:hidden">
              <div style="padding:0.9rem 1.25rem;font-weight:500;font-size:14px;border-bottom:0.5px solid var(--border);background:var(--success-bg);color:var(--success-fg)">
              ✅ Produk Anda
            </div>
            <div style="padding:0.75rem 1rem;display:flex;flex-direction:column;gap:8px">
              {#each data.delivered_items as item, i}
                <div style="display:flex;align-items:flex-start;gap:10px;padding:10px 12px;background:var(--surface-2);border-radius:var(--radius)">
                  <div style="width:22px;height:22px;border-radius:50%;background:var(--primary);color:var(--primary-fg);font-size:11px;font-weight:600;display:flex;align-items:center;justify-content:center;flex-shrink:0;margin-top:2px">{i+1}</div>
                  {#if isURL(item)}
                    <div style="flex:1;min-width:0">
                      <a href={item} target="_blank" rel="noopener"
                        style="display:inline-flex;align-items:center;gap:6px;background:linear-gradient(135deg,var(--primary),var(--primary-2));color:var(--primary-fg);padding:8px 16px;border-radius:6px;font-size:13px;font-weight:500;text-decoration:none">
                        📥 Download
                      </a>
                      <div class="mono" style="font-size:10.5px;color:var(--text-muted);margin-top:5px;word-break:break-all">{item}</div>
                    </div>
                  {:else}
                    <div style="flex:1">
                      <div style="font-size:11px;color:var(--text-muted);margin-bottom:4px">Nomer & Pin Untuk Login / Email & Password  :</div>
                      <code style="display:block;font-family:'JetBrains Mono',monospace;font-size:13px;background:var(--surface);border:0.5px solid var(--border);padding:8px 12px;border-radius:6px;word-break:break-all;user-select:all">{item}</code>
                    </div>
                  {/if}
                </div>
              {/each}
            </div>
            <div style="padding:0.75rem 1.25rem;background:var(--success-bg);border-top:0.5px solid var(--border);font-size:12px;color:var(--success-fg)">
              💡 Item juga dikirim ke email Anda.
            </div>
          </div>

        {:else if statusKey === 'waiting_payment' || statusKey === 'verifying'}
          <div class="card" style="text-align:center;padding:2.5rem">
            <div class="pulse">⏳</div>
            <div style="font-weight:500;font-size:14px;margin:12px 0 6px">Menunggu Konfirmasi</div>
            <div style="font-size:13px;color:var(--text-muted)">Produk muncul otomatis setelah bayar.</div>
          </div>

        {:else if data.status === 'script_executed'}
          <div class="card" style="text-align:center;padding:2.5rem">
            <div style="font-size:40px;margin-bottom:12px">👷</div>
            <div style="font-weight:500;font-size:14px;margin-bottom:6px">Sedang Diproses</div>
            <div style="font-size:13px;color:var(--text-muted)">Tim kami menghubungi dalam 1×24 jam.</div>
          </div>

        {:else if ['expired','cancelled','failed'].includes(statusKey)}
          <div class="card" style="text-align:center;padding:2.5rem">
            <div style="font-size:40px;margin-bottom:12px">❌</div>
            <div style="font-weight:500;font-size:14px;margin-bottom:12px">
              {statusKey === 'expired' ? 'Waktu Pembayaran Habis' : 'Pesanan Tidak Berhasil'}
            </div>
            <a href="/" class="btn btn-primary">Beli Lagi</a>
          </div>
        {/if}

        <div style="margin-top:12px;text-align:center">
          <a href="/cek-invoice?no={invoiceNo}" style="font-size:12.5px;color:var(--text-muted)">
            Cek invoice dengan email →
          </a>
        </div>

        <div style="margin-top:8px;text-align:center">
          <a href="/komplain?invoice={invoiceNo}" style="font-size:12.5px;color:var(--primary);font-weight:500">
            Komplain ke WhatsApp Admin →
          </a>
        </div>
      </div>
    </div>
  {/if}
</div>

<style>
.invoice-nav {
  background: var(--surface);
  border-bottom: 0.5px solid var(--border);
  padding: 0 1.5rem;
  position: sticky;
  top: 0;
  z-index: 100;
  backdrop-filter: blur(14px);
}

.invoice-nav-inner {
  max-width: 800px;
  margin: 0 auto;
  min-height: 54px;
  display: flex;
  align-items: center;
  gap: 10px;
  flex-wrap: wrap;
  padding: 8px 0;
}

.invoice-brand {
  display: flex;
  align-items: center;
  gap: 8px;
  font-weight: 500;
  font-size: 15px;
}

.invoice-actions {
  display: flex;
  align-items: center;
  gap: 8px;
  margin-left: auto;
  flex-wrap: wrap;
  justify-content: flex-end;
}

.invoice-actions :global(.btn) {
  white-space: nowrap;
}

.invoice-meta {
  font-size: 12px;
  color: var(--text-muted);
}

.pay-grid { display:grid;grid-template-columns:1fr 1fr;gap:14px; }
@media(max-width:640px) { .pay-grid { grid-template-columns:1fr; } }
.info-box { background:var(--info-bg);border-radius:var(--radius);padding:10px 14px;font-size:13px;color:var(--info-fg); }
.pulse { font-size:40px;display:inline-block;animation:pulse 2s ease-in-out infinite; }
@keyframes pulse { 0%,100%{transform:scale(1);opacity:1} 50%{transform:scale(1.15);opacity:0.7} }

.detail-toggle {
  display: none;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
  width: 100%;
  padding: 11px 14px;
  border: 1px solid var(--border);
  border-radius: var(--radius);
  background: var(--surface);
  color: var(--text);
  cursor: pointer;
  box-shadow: var(--shadow);
}

.detail-toggle span:last-child {
  font-size: 12px;
  color: var(--text-muted);
}

.detail-panel {
  display: block;
}

.detail-head {
  padding: 0.9rem 1.25rem;
  font-weight: 500;
  font-size: 14px;
  border-bottom: 0.5px solid var(--border);
}

.detail-table {
  width: 100%;
  border-collapse: collapse;
}

.detail-table td {
  padding: 8px 16px;
  font-size: 13px;
}

.detail-table td:first-child {
  color: var(--text-muted);
  width: 38%;
}

.detail-table td:last-child {
  text-align: right;
}

.detail-total {
  border-top: 2px solid var(--primary);
}

.detail-total td {
  padding: 11px 16px;
}

.detail-total td:first-child {
  font-weight: 600;
  color: var(--text);
}

.detail-total td:last-child {
  font-weight: 700;
  font-size: 19px;
  color: var(--primary);
}

@media (max-width: 640px) {
  .invoice-nav { padding-inline: 0.75rem; }
  .invoice-nav-inner { align-items: flex-start; }
  .invoice-actions { width: 100%; margin-left: 0; justify-content: flex-start; }
  .invoice-actions :global(.btn), .invoice-actions :global(.nav-action-btn) { flex: 1 1 calc(50% - 4px); }
  .invoice-meta { width: 100%; order: 3; }
  .detail-toggle { display: inline-flex; }
  .detail-panel:not(.detail-open) { display: none; }
}
</style>

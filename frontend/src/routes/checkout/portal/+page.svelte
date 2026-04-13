<script>
  import { onMount, onDestroy } from 'svelte';
  import { page } from '$app/stores';
  import { goto } from '$app/navigation';
  import { api } from '$lib/api.js';

  const invoiceNo  = $page.url.searchParams.get('invoice') || '';
  const payURL     = $page.url.searchParams.get('url')     || '';
  const credParam  = $page.url.searchParams.get('cred')    || ''; // token atau email

  // Ambil credential dari sessionStorage jika tidak ada di URL
  const cred = credParam ||
    (typeof sessionStorage !== 'undefined' ? sessionStorage.getItem('inv_token_' + invoiceNo) || '' : '');

  let orderStatus  = 'pending';
  let loading      = true;
  let pollInterval = null;
  let portalWindow = null;
  let portalClosed = false;
  let countdown    = 4;
  let countdownTimer = null;
  let checkCount   = 0;
  let isMobile     = false;
  let mobileAutoOpened = false;

  async function checkStatus() {
    if (!invoiceNo) return;
    try {
      const data = await api.getInvoice(invoiceNo, cred);
      orderStatus = data.status;
      checkCount++;
      loading = false;
      if (isPaid) { stopPolling(); setTimeout(() => goto(`/payment/${invoiceNo}`), 1000); }
      if (isFinal && !isPaid) stopPolling();
    } catch {}
  }

  function startPolling() {
    checkStatus();
    pollInterval = setInterval(checkStatus, 4000);
    countdownTimer = setInterval(() => {
      countdown = isPaid || isFinal ? 0 : (countdown <= 1 ? 4 : countdown - 1);
    }, 1000);
  }
  function stopPolling() {
    clearInterval(pollInterval);
    clearInterval(countdownTimer);
  }

  function openPayment() {
    portalClosed = false;
    if (isMobile) {
      // Mobile: redirect langsung ke URL (payment gateway biasanya support mobile web)
      // Browser akan otomatis membuka app jika tersedia, atau mobile web jika tidak
      window.location.href = payURL;
    } else {
      // Desktop: buka di tab baru
      portalWindow = window.open(payURL, '_blank', 'noopener,noreferrer');
      if (portalWindow) {
        const watcher = setInterval(() => {
          if (portalWindow.closed) {
            portalClosed = true;
            clearInterval(watcher);
            checkStatus();
          }
        }, 800);
      }
    }
  }

  onMount(() => {
    if (!invoiceNo || !payURL) { goto('/'); return; }

    // Deteksi mobile
    isMobile = /Android|iPhone|iPad|iPod|Mobile/i.test(navigator.userAgent);

    if (isMobile && !mobileAutoOpened) {
      // Mobile: redirect otomatis ke URL pembayaran
      mobileAutoOpened = true;
      setTimeout(() => {
        window.location.href = payURL;
      }, 600); // delay 600ms biar halaman ini sempat render dulu
    } else {
      // Desktop: buka tab baru
      openPayment();
    }

    startPolling();
  });
  onDestroy(stopPolling);

  $: isPaid  = orderStatus === 'paid' || orderStatus === 'script_executed';
  $: isFinal = isPaid || ['expired','cancelled','failed'].includes(orderStatus);
</script>

<svelte:head><title>Portal Pembayaran — Digitalkuh Murah</title></svelte:head>

<div class="portal-wrap">

  {#if isPaid}
    <!-- SUKSES -->
    <div class="portal-card">
      <div class="success-anim">✅</div>
      <h2 style="font-size:20px;font-weight:500;margin:14px 0 6px;color:#2f5e0f">Pembayaran Berhasil!</h2>
      <p style="font-size:13.5px;color:var(--text-muted);margin-bottom:18px">Produk Anda sedang dikirim otomatis.</p>
      <div class="invoice-badge">{invoiceNo}</div>
      <p style="font-size:12px;color:var(--text-muted);margin-top:10px">Mengalihkan ke halaman produk…</p>
    </div>

  {:else if ['expired','cancelled','failed'].includes(orderStatus)}
    <!-- GAGAL -->
    <div class="portal-card">
      <div style="font-size:48px;margin-bottom:14px">❌</div>
      <h2 style="font-size:18px;font-weight:500;margin-bottom:8px;color:#8c2626">
        {orderStatus === 'expired' ? 'Waktu Pembayaran Habis' : 'Pembayaran Gagal'}
      </h2>
      <p style="font-size:13px;color:var(--text-muted);margin-bottom:20px">Pesanan dibatalkan.</p>
      <a href="/" class="btn btn-primary">Kembali ke Toko</a>
    </div>

  {:else if isMobile}
    <!-- MOBILE: sedang membuka app pembayaran -->
    <div class="portal-card">
      <div class="brand">
        <span class="brand-logo">🛍</span>
        <span>Digitalkuh Murah</span>
      </div>

      {#if mobileAutoOpened}
        <div style="font-size:40px;margin-bottom:14px;animation:spin 1.5s linear infinite">🔄</div>
        <div style="font-weight:500;font-size:15px;margin-bottom:8px">Membuka halaman pembayaran…</div>
        <p style="font-size:13px;color:var(--text-muted);margin-bottom:20px">
          Jika tidak otomatis terbuka, tap tombol di bawah ini.
        </p>
      {:else}
        <div style="font-size:13px;color:var(--text-muted);margin-bottom:16px">
          Selesaikan pembayaran lalu kembali ke halaman ini.
        </div>
      {/if}

      <div class="invoice-badge" style="margin-bottom:16px">{invoiceNo}</div>

      <a href={payURL} class="btn btn-primary" style="display:block;padding:13px;font-size:15px;text-decoration:none;margin-bottom:10px">
        💳 Bayar Sekarang
      </a>

      <div class="check-row" style="margin-top:14px">
        <span style="font-size:12px;color:var(--text-muted)">Sudah bayar? Cek dalam {countdown}s</span>
        <button class="btn btn-sm" on:click={checkStatus}>🔄 Cek</button>
      </div>

      <div style="margin-top:12px;text-align:center">
        <a href="/payment/{invoiceNo}" style="font-size:12px;color:var(--text-muted)">
          Lihat status invoice →
        </a>
      </div>
    </div>

  {:else}
    <!-- DESKTOP: tab baru sudah dibuka -->
    <div class="portal-card">
      <div class="brand">
        <span class="brand-logo">🛍</span>
        <span>Digitalkuh Murah</span>
      </div>

      <div class="status-ring">
        <div class="pulse-dot"></div>
        <span>Menunggu Pembayaran</span>
      </div>

      <div class="invoice-badge">{invoiceNo}</div>

      <div class="instruction-box">
        {#if portalClosed}
          <div style="font-size:13px;font-weight:500;color:#854F0B;margin-bottom:6px">⚠️ Tab pembayaran ditutup</div>
          <p style="font-size:13px;color:var(--text-muted)">Jika sudah membayar, sistem akan mendeteksi otomatis. Atau buka kembali.</p>
          <button class="btn btn-primary" style="margin-top:10px;font-size:13px" on:click={openPayment}>🔄 Buka Lagi</button>
        {:else}
          <div style="font-size:13px;font-weight:500;margin-bottom:6px">Tab pembayaran sudah terbuka</div>
          <p style="font-size:12.5px;color:var(--text-muted)">Selesaikan pembayaran di tab tersebut. Halaman ini otomatis diperbarui.</p>
        {/if}
      </div>

      <div class="check-row">
        <span style="font-size:12px;color:var(--text-muted)">Auto-cek dalam <strong>{countdown}s</strong> · {checkCount}× dicek</span>
        <button class="btn btn-sm" style="font-size:11px" on:click={checkStatus}>🔄 Cek</button>
      </div>

      <div style="margin-top:14px;text-align:center">
        <a href="/payment/{invoiceNo}" style="font-size:12px;color:var(--text-muted)">
          Lihat status invoice →
        </a>
      </div>
    </div>
  {/if}
</div>

<style>
.portal-wrap { min-height:100vh;display:flex;align-items:center;justify-content:center;background:linear-gradient(135deg,#f0f6fd 0%,#f8f8f6 100%);padding:1rem; }
.portal-card { background:#fff;border:0.5px solid var(--border);border-radius:var(--radius-lg);padding:2.5rem 2rem;max-width:420px;width:100%;text-align:center;box-shadow:0 4px 24px rgba(0,0,0,0.06); }
.brand { display:flex;align-items:center;justify-content:center;gap:8px;font-weight:500;font-size:15px;margin-bottom:1.5rem; }
.brand-logo { background:#0d5fa8;border-radius:8px;width:28px;height:28px;display:flex;align-items:center;justify-content:center;font-size:14px; }
.status-ring { display:inline-flex;align-items:center;gap:8px;background:#FAEEDA;color:#854F0B;padding:8px 18px;border-radius:999px;font-size:13.5px;font-weight:500;margin-bottom:16px; }
.pulse-dot { width:8px;height:8px;border-radius:50%;background:#854F0B;animation:blink 1.4s ease-in-out infinite; }
@keyframes blink { 0%,100%{opacity:1} 50%{opacity:0.3} }
.invoice-badge { display:inline-block;font-family:'JetBrains Mono',monospace;font-size:12.5px;background:#f0f4ff;color:#0d5fa8;padding:5px 14px;border-radius:999px;margin-bottom:16px; }
.instruction-box { background:#f8f8f6;border-radius:var(--radius);padding:14px 16px;margin-bottom:14px;text-align:left; }
.check-row { display:flex;align-items:center;justify-content:space-between;gap:8px;padding:8px 12px;background:#f0f4ff;border-radius:var(--radius); }
.success-anim { font-size:56px;animation:pop 0.5s cubic-bezier(.36,.07,.19,.97); }
@keyframes pop { 0%{transform:scale(0.5);opacity:0} 80%{transform:scale(1.1)} 100%{transform:scale(1);opacity:1} }
@keyframes spin { to { transform:rotate(360deg); } }
</style>

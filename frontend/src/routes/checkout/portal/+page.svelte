<script>
  import { onMount, onDestroy } from 'svelte';
  import { page } from '$app/stores';
  import { goto } from '$app/navigation';
  import { api } from '$lib/api.js';

  const invoiceNo  = $page.url.searchParams.get('invoice') || '';
  const payURL     = $page.url.searchParams.get('url')     || '';
  const credParam  = $page.url.searchParams.get('cred')    || '';

  const cred = credParam ||
    (typeof sessionStorage !== 'undefined' ? sessionStorage.getItem('inv_token_' + invoiceNo) || '' : '');

  let orderStatus = 'pending';
  let loading     = true;
  let pollInterval = null;
  let portalClosed = false;
  let countdown    = 4;
  let countdownTimer = null;
  let checkCount   = 0;
  let isMobile     = false;
  let paymentOpened = false;  // sudah buka tab/halaman bayar?

  $: isPaid  = orderStatus === 'paid' || orderStatus === 'script_executed';
  $: isFailed = ['expired','cancelled','failed'].includes(orderStatus);
  $: isFinal  = isPaid || isFailed;

  async function checkStatus() {
    if (!invoiceNo) return;
    try {
      const data = await api.getInvoice(invoiceNo, cred);
      orderStatus = data.status;
      checkCount++;
      loading = false;
      if (isPaid) {
        stopPolling();
        // Redirect ke halaman invoice setelah 1.2 detik
        setTimeout(() => goto(`/payment/${data.invoice_no || invoiceNo}`), 1200);
      }
      if (isFailed) stopPolling();
    } catch {}
  }

  function startPolling() {
    checkStatus();
    pollInterval = setInterval(checkStatus, 4000);
    countdownTimer = setInterval(() => {
      if (isFinal) { countdown = 0; return; }
      countdown = countdown <= 1 ? 4 : countdown - 1;
    }, 1000);
  }
  function stopPolling() {
    clearInterval(pollInterval);
    clearInterval(countdownTimer);
  }

  // Buka URL pembayaran di tab baru — berlaku untuk desktop DAN mobile
  function openPaymentTab() {
    portalClosed = false;
    paymentOpened = true;
    const win = window.open(payURL, '_blank', 'noopener,noreferrer');

    // Pantau jika tab ditutup (hanya bisa di desktop, mobile tidak bisa)
    if (win) {
      const watcher = setInterval(() => {
        try {
          if (win.closed) {
            portalClosed = true;
            clearInterval(watcher);
            // Langsung cek status begitu tab ditutup
            checkStatus();
          }
        } catch {}
      }, 800);
    }
  }

  onMount(() => {
    if (!invoiceNo || !payURL) { goto('/'); return; }

    // Deteksi mobile
    isMobile = /Android|iPhone|iPad|iPod|Mobile/i.test(
      typeof navigator !== 'undefined' ? navigator.userAgent : ''
    );

    // Langsung buka tab pembayaran — baik mobile maupun desktop
    openPaymentTab();
    startPolling();
  });
  onDestroy(stopPolling);
</script>

<svelte:head><title>Portal Pembayaran — DigiStore</title></svelte:head>

<div class="portal-wrap">
  {#if isPaid}
    <!-- ── SUKSES ─────────────────────────── -->
    <div class="portal-card">
      <div class="success-anim">✅</div>
      <h2 style="font-size:20px;font-weight:500;margin:14px 0 6px;color:#2f5e0f">Pembayaran Berhasil!</h2>
      <p style="font-size:13.5px;color:var(--text-muted);margin-bottom:18px">Produk sedang dikirim otomatis.</p>
      <div class="invoice-badge">{invoiceNo}</div>
      <p style="font-size:12px;color:var(--text-muted);margin-top:10px">Mengalihkan…</p>
    </div>

  {:else if isFailed}
    <!-- ── GAGAL ──────────────────────────── -->
    <div class="portal-card">
      <div style="font-size:48px;margin-bottom:14px">❌</div>
      <h2 style="font-size:18px;font-weight:500;margin-bottom:8px;color:#8c2626">
        {orderStatus === 'expired' ? 'Waktu Bayar Habis' : 'Pembayaran Gagal'}
      </h2>
      <p style="font-size:13px;color:var(--text-muted);margin-bottom:20px">
        Pesanan dibatalkan otomatis.
      </p>
      <a href="/" class="btn btn-primary">Kembali ke Toko</a>
    </div>

  {:else}
    <!-- ── MENUNGGU ────────────────────────── -->
    <div class="portal-card">
      <div class="brand">
        <span class="brand-logo">🛍</span>
        <span>DigiStore</span>
      </div>

      <div class="status-ring">
        <div class="pulse-dot"></div>
        <span>Menunggu Pembayaran</span>
      </div>

      <div class="invoice-badge">{invoiceNo}</div>

      {#if loading}
        <div style="color:var(--text-muted);font-size:13px;margin-bottom:16px">Memeriksa status…</div>
      {:else if portalClosed}
        <!-- Tab sudah ditutup pengguna -->
        <div class="instruction-box" style="text-align:left">
          <div style="font-size:13px;font-weight:500;color:#854F0B;margin-bottom:6px">
            ⚠️ Halaman pembayaran ditutup
          </div>
          <p style="font-size:13px;color:var(--text-muted);margin-bottom:10px">
            Jika sudah membayar, sistem akan mendeteksi otomatis dalam beberapa detik.
            Atau buka kembali halaman bayar.
          </p>
          <button class="btn btn-primary" style="width:100%;padding:11px;font-size:14px"
            on:click={openPaymentTab}>
            🔗 Buka Halaman Bayar di Tab Baru
          </button>
        </div>

      {:else if paymentOpened}
        <!-- Tab sudah terbuka -->
        <div class="instruction-box" style="text-align:left">
          {#if isMobile}
            <div style="font-size:13px;font-weight:500;margin-bottom:6px">
              📱 Halaman pembayaran dibuka di tab baru
            </div>
            <p style="font-size:12.5px;color:var(--text-muted)">
              Pindah ke tab pembayaran, selesaikan transaksi, lalu kembali ke tab ini.
              Status akan diperbarui otomatis.
            </p>
          {:else}
            <div style="font-size:13px;font-weight:500;margin-bottom:6px">
              💻 Halaman pembayaran sudah terbuka di tab baru
            </div>
            <p style="font-size:12.5px;color:var(--text-muted)">
              Selesaikan pembayaran di tab tersebut.
              Halaman ini otomatis diperbarui setelah terkonfirmasi.
            </p>
          {/if}
          <button class="btn btn-sm" style="margin-top:10px;width:100%" on:click={openPaymentTab}>
            🔗 Tidak terbuka? Klik di sini
          </button>
        </div>
      {/if}

      <!-- Auto cek countdown -->
      <div class="check-row">
        <span style="font-size:12px;color:var(--text-muted)">
          Cek otomatis dalam <strong>{countdown}s</strong> · sudah {checkCount}×
        </span>
        <button class="btn btn-sm" style="font-size:11px" on:click={checkStatus}>🔄 Cek</button>
      </div>

      <div style="margin-top:12px;text-align:center">
        <a href="/payment/{invoiceNo}" style="font-size:12px;color:var(--text-muted)">
          Lihat status invoice →
        </a>
      </div>
    </div>
  {/if}
</div>

<style>
.portal-wrap {
  min-height: 100vh;
  display: flex; align-items: center; justify-content: center;
  background: linear-gradient(135deg, #f0f6fd 0%, #f8f8f6 100%);
  padding: 1rem;
}
.portal-card {
  background: #fff;
  border: 0.5px solid var(--border);
  border-radius: var(--radius-lg);
  padding: 2.5rem 2rem;
  max-width: 420px; width: 100%;
  text-align: center;
  box-shadow: 0 4px 24px rgba(0,0,0,0.06);
}
.brand {
  display: flex; align-items: center; justify-content: center;
  gap: 8px; font-weight: 500; font-size: 15px; margin-bottom: 1.5rem;
}
.brand-logo {
  background: #0d5fa8; border-radius: 8px;
  width: 28px; height: 28px;
  display: flex; align-items: center; justify-content: center; font-size: 14px;
}
.status-ring {
  display: inline-flex; align-items: center; gap: 8px;
  background: #FAEEDA; color: #854F0B;
  padding: 8px 18px; border-radius: 999px;
  font-size: 13.5px; font-weight: 500; margin-bottom: 16px;
}
.pulse-dot {
  width: 8px; height: 8px; border-radius: 50%;
  background: #854F0B;
  animation: blink 1.4s ease-in-out infinite;
}
@keyframes blink { 0%,100%{opacity:1} 50%{opacity:0.3} }
.invoice-badge {
  display: inline-block;
  font-family: 'JetBrains Mono', monospace; font-size: 12.5px;
  background: #f0f4ff; color: #0d5fa8;
  padding: 5px 14px; border-radius: 999px; margin-bottom: 16px;
}
.instruction-box {
  background: #f8f8f6;
  border-radius: var(--radius);
  padding: 14px 16px; margin-bottom: 14px;
}
.check-row {
  display: flex; align-items: center; justify-content: space-between;
  gap: 8px; padding: 8px 12px;
  background: #f0f4ff; border-radius: var(--radius);
}
.success-anim {
  font-size: 56px;
  animation: pop 0.5s cubic-bezier(.36,.07,.19,.97);
}
@keyframes pop {
  0%{transform:scale(0.5);opacity:0}
  80%{transform:scale(1.1)}
  100%{transform:scale(1);opacity:1}
}
</style>

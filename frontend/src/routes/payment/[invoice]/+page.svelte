<script>
  import { onMount, onDestroy } from 'svelte';
  import { page } from '$app/stores';
  import { api } from '$lib/api.js';
  import { IDR, fmtDateTime, PAY_LABEL } from '$lib/utils.js';

  const invoiceNo = $page.params.invoice;

  let data = null;
  let loading = true;
  let error = '';
  let pollInterval = null;
  let countdown = 8;
  let countdownTimer = null;
  let pollCount = 0;
  let lastChecked = null;

  $: isFinal = data && ['paid','cancelled','expired','failed'].includes(data.status);
  $: gatewayActive = data?.gateway_pay_url || data?.gateway_pay_code;
  function isURL(s) { return s?.startsWith('http'); }

  async function fetchStatus() {
    try {
      data = await api.getInvoice(invoiceNo);
      pollCount++; lastChecked = new Date(); error = '';
      if (isFinal) stopPolling();
    } catch(e) { error = e.message; }
    finally { loading = false; }
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

  onMount(startPolling);
  onDestroy(stopPolling);

  const SC = {
    paid:            { icon:'✅', color:'#2f5e0f', bg:'#EAF3DE', title:'Pembayaran Dikonfirmasi!',   desc:'Produk Anda sudah siap di bawah.' },
    pending:         { icon:'⏳', color:'#854F0B', bg:'#FAEEDA', title:'Menunggu Pembayaran',        desc:'Halaman ini diperbarui otomatis.' },
    script_executed: { icon:'⚙️', color:'#185FA5', bg:'#E6F1FB', title:'Pesanan Diproses',          desc:'Tim akan menghubungi dalam 1×24 jam.' },
    expired:         { icon:'⌛', color:'#8c2626', bg:'#FCEBEB', title:'Waktu Pembayaran Habis',    desc:'Pesanan otomatis dibatalkan.' },
    failed:          { icon:'❌', color:'#8c2626', bg:'#FCEBEB', title:'Pembayaran Gagal',          desc:'Silakan coba lagi.' },
    cancelled:       { icon:'✗',  color:'#8c2626', bg:'#FCEBEB', title:'Pesanan Dibatalkan',        desc:'Hubungi kami jika ada pertanyaan.' },
  };

  // Hitung sisa waktu expired
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

<svelte:head><title>Pembayaran {invoiceNo} — Digitalku Murah</title></svelte:head>

<nav style="background:#fff;border-bottom:0.5px solid var(--border);padding:0 1.5rem;position:sticky;top:0;z-index:100">
  <div style="max-width:800px;margin:0 auto;height:54px;display:flex;align-items:center;gap:10px">
    <a href="/" style="display:flex;align-items:center;gap:8px;font-weight:500;font-size:15px">
      <span style="background:#0d5fa8;border-radius:8px;width:28px;height:28px;display:flex;align-items:center;justify-content:center;font-size:14px">🛍</span>
      Digitalku Murah
    </a>
    <span class="mono" style="margin-left:auto;font-size:12px;color:var(--text-muted)">{invoiceNo}</span>
  </div>
</nav>

<div style="max-width:780px;margin:0 auto;padding:1.5rem 1rem 3rem">
  {#if loading}
    <div style="text-align:center;padding:4rem;color:var(--text-muted)">
      <div style="font-size:32px;margin-bottom:10px">🔄</div>Memuat…
    </div>

  {:else if error}
    <div class="card" style="text-align:center;padding:2.5rem">
      <div style="font-size:32px;margin-bottom:10px">❌</div>
      <div style="font-weight:500;margin-bottom:16px">{error}</div>
      <a href="/" class="btn btn-primary">Kembali ke Toko</a>
    </div>

  {:else if data}
    {@const sc = SC[data.status] || SC.pending}

    <!-- Status banner -->
    <div style="background:{sc.bg};border-radius:var(--radius-lg);padding:1.25rem 1.5rem;margin-bottom:16px;display:flex;align-items:center;justify-content:space-between;gap:12px">
      <div style="display:flex;align-items:center;gap:14px">
        <span style="font-size:36px">{sc.icon}</span>
        <div>
          <div style="font-weight:500;font-size:17px;color:{sc.color}">{sc.title}</div>
          <div style="font-size:13px;color:{sc.color};opacity:0.8;margin-top:3px">{sc.desc}</div>
        </div>
      </div>

      {#if data.status === 'pending' && !isFinal}
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
      <!-- Kiri: detail + instruksi bayar -->
      <div style="display:flex;flex-direction:column;gap:12px">

        <!-- Detail order -->
        <div class="card" style="padding:0;overflow:hidden">
          <div style="padding:0.9rem 1.25rem;font-weight:500;font-size:14px;border-bottom:0.5px solid var(--border)">Detail Pesanan</div>
          <table style="width:100%;border-collapse:collapse">
            {#each [
              ['Produk', data.product_name],
              ['Pembeli', data.buyer_name],
              ['Jumlah', `${data.qty} pcs`],
              ['Metode', PAY_LABEL[data.pay_method] || data.pay_method],
            ] as [l,v], i}
             <tbody>
               <tr style="background:{i%2===0?'#f9f9f9':'#fff'}">
                <td style="padding:8px 16px;font-size:12px;color:var(--text-muted);width:38%">{l}</td>
                <td style="padding:8px 16px;font-size:13px;text-align:right">{v}</td>
              </tr>
             </tbody>
            {/each}
            <tbody>
              <tr style="border-top:2px solid #0d5fa8">
                <td style="padding:11px 16px;font-weight:600">Total</td>
                <td style="padding:11px 16px;text-align:right;font-weight:700;font-size:19px;color:#0d5fa8">{IDR(data.total)}</td>
              </tr>
            </tbody>
          </table>
        </div>

        <!-- Tombol bayar via DompetX -->
        {#if data.status === 'pending' && data.gateway_pay_url}
          <div class="card" style="text-align:center;padding:1.5rem">
            <div style="font-weight:500;font-size:14px;margin-bottom:8px">Selesaikan Pembayaran</div>
            <a href={data.gateway_pay_url} target="_blank" rel="noopener"
              style="display:inline-block;background:#0d5fa8;color:#fff;padding:13px 28px;border-radius:8px;font-size:15px;font-weight:600;text-decoration:none">
              💳 Bayar via DompetX →
            </a>
            <div style="font-size:12px;color:var(--text-muted);margin-top:8px">
              Halaman pembayaran akan terbuka di tab baru. Kembali ke halaman ini setelah selesai.
            </div>
            {#if data.expired_at}
              <div style="font-size:12px;color:#854F0B;margin-top:4px">
                ⚠️ Berlaku sampai: {fmtDateTime(data.expired_at)}
              </div>
            {/if}
          </div>

        {:else if data.status === 'pending' && data.gateway_pay_code}
          <!-- Tampilkan nomor VA jika tidak ada payment_url tapi ada pay_code -->
          <div class="card">
            <div style="font-weight:500;font-size:14px;margin-bottom:10px">Nomor Virtual Account / Kode Bayar</div>
            <div style="background:#f0f4ff;border-radius:var(--radius);padding:14px;text-align:center">
              <div style="font-size:11px;color:var(--text-muted);margin-bottom:5px">{PAY_LABEL[data.pay_method] || data.pay_method}</div>
              <div class="mono" style="font-size:24px;font-weight:700;letter-spacing:3px">{data.gateway_pay_code}</div>
            </div>
            {#if data.expired_at}
              <div style="font-size:12px;color:#854F0B;margin-top:8px">
                ⚠️ Berlaku sampai: {fmtDateTime(data.expired_at)}
              </div>
            {/if}
          </div>

        {:else if data.status === 'pending' && !gatewayActive}
          <!-- Mode manual: tampilkan instruksi transfer -->
          <div class="card">
            <div style="font-weight:500;font-size:14px;margin-bottom:10px">📋 Instruksi Pembayaran Manual</div>
            <div style="font-size:13px;color:var(--text-muted);margin-bottom:10px">
              Lakukan transfer dan konfirmasikan ke admin. Produk akan dikirim setelah pembayaran dikonfirmasi.
            </div>
            <div style="background:#f8f8f6;border-radius:var(--radius);padding:12px;font-size:13px">
              Sertakan nomor invoice <strong class="mono">{invoiceNo}</strong> sebagai keterangan transfer.
            </div>
          </div>
        {/if}

        <!-- Last checked -->
        {#if lastChecked && !isFinal}
          <div style="font-size:11.5px;color:var(--text-hint);text-align:center">
            Dicek {pollCount}× · terakhir {fmtDateTime(lastChecked.toISOString())}
          </div>
        {/if}
      </div>

      <!-- Kanan: produk / status proses -->
      <div>
        {#if data.status === 'paid' && data.delivered_items?.length}
          <div class="card" style="padding:0;overflow:hidden">
            <div style="padding:0.9rem 1.25rem;font-weight:500;font-size:14px;border-bottom:0.5px solid var(--border);background:#EAF3DE;color:#2f5e0f">
              ✅ Produk Anda
            </div>
            <div style="padding:0.75rem 1rem;display:flex;flex-direction:column;gap:8px">
              {#each data.delivered_items as item, i}
                <div style="display:flex;align-items:flex-start;gap:10px;padding:10px 12px;background:#f8f8f6;border-radius:var(--radius)">
                  <div style="width:22px;height:22px;border-radius:50%;background:#0d5fa8;color:#fff;font-size:11px;font-weight:600;display:flex;align-items:center;justify-content:center;flex-shrink:0;margin-top:2px">{i+1}</div>
                  {#if isURL(item)}
                    <div style="flex:1;min-width:0">
                      <a href={item} target="_blank" rel="noopener"
                        style="display:inline-flex;align-items:center;gap:6px;background:#0d5fa8;color:#fff;padding:8px 16px;border-radius:6px;font-size:13px;font-weight:500;text-decoration:none">
                        📥 Download
                      </a>
                      <div class="mono" style="font-size:10.5px;color:var(--text-muted);margin-top:5px;word-break:break-all">{item}</div>
                    </div>
                  {:else}
                    <div style="flex:1">
                      <div style="font-size:11px;color:var(--text-muted);margin-bottom:4px">License Key:</div>
                      <code style="display:block;font-family:'JetBrains Mono',monospace;font-size:13px;background:#fff;border:0.5px solid var(--border);padding:8px 12px;border-radius:6px;word-break:break-all;user-select:all">{item}</code>
                    </div>
                  {/if}
                </div>
              {/each}
            </div>
            <div style="padding:0.75rem 1.25rem;background:#f0f8e8;border-top:0.5px solid var(--border);font-size:12px;color:#3B6D11">
              💡 Item juga dikirim ke email Anda.
            </div>
          </div>

        {:else if data.status === 'pending'}
          <div class="card" style="text-align:center;padding:2.5rem">
            <div class="pulse">⏳</div>
            <div style="font-weight:500;font-size:14px;margin:12px 0 6px">Menunggu Konfirmasi</div>
            <div style="font-size:13px;color:var(--text-muted)">Produk akan muncul di sini otomatis setelah bayar.</div>
          </div>

        {:else if data.status === 'script_executed'}
          <div class="card" style="text-align:center;padding:2.5rem">
            <div style="font-size:40px;margin-bottom:12px">👷</div>
            <div style="font-weight:500;font-size:14px;margin-bottom:6px">Sedang Diproses</div>
            <div style="font-size:13px;color:var(--text-muted)">Tim kami akan menghubungi Anda dalam 1×24 jam.</div>
          </div>

        {:else if ['expired','cancelled','failed'].includes(data.status)}
          <div class="card" style="text-align:center;padding:2.5rem">
            <div style="font-size:40px;margin-bottom:12px">❌</div>
            <div style="font-weight:500;font-size:14px;margin-bottom:12px">
              {data.status === 'expired' ? 'Waktu Pembayaran Habis' : 'Pesanan Tidak Berhasil'}
            </div>
            <a href="/" class="btn btn-primary">Beli Lagi</a>
          </div>
        {/if}

        <div style="margin-top:12px;text-align:center">
          <a href="/cek-invoice?no={invoiceNo}" style="font-size:12.5px;color:var(--text-muted)">
            Buka halaman cek invoice →
          </a>
        </div>
      </div>
    </div>
  {/if}
</div>

<style>
.pay-grid { display:grid;grid-template-columns:1fr 1fr;gap:14px; }
@media(max-width:640px) { .pay-grid { grid-template-columns:1fr; } }
.pulse { font-size:40px;display:inline-block;animation:pulse 2s ease-in-out infinite; }
@keyframes pulse { 0%,100%{transform:scale(1);opacity:1} 50%{transform:scale(1.15);opacity:0.7} }
</style>

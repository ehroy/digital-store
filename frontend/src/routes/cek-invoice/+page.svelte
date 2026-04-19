<script>
  import { onMount } from 'svelte';
  import { page } from '$app/stores';
  import { api } from '$lib/api.js';
  import { IDR, fmtDateTime, STATUS_LABEL, PAY_LABEL } from '$lib/utils.js';
  import ThemeToggle from '$lib/ThemeToggle.svelte';
  import { toDataURL } from 'qrcode';

  let invoiceNo = '';
  let email = '';
  let result = null;
  let loading = false;
  let error = '';
  let requireEmail = false;
  let qrisDataUrl = '';
  let credential = '';

  $: statusKey = result?.status === 'pending' ? 'waiting_payment' : result?.status;
  $: paymentLink = result?.gateway_redirect_url || result?.gateway_pay_url || '';
  $: qrisString = result?.gateway_qris_string || result?.gateway_pay_code || '';
  $: qrisImage = result?.gateway_qris_image_url || '';

  onMount(() => {
    const no = $page.url.searchParams.get('no');
    const em = $page.url.searchParams.get('email');
    const cred = $page.url.searchParams.get('cred');
    if (no) invoiceNo = no;
    if (em) email = em;
    credential = cred || (no && typeof sessionStorage !== 'undefined'
      ? sessionStorage.getItem('inv_token_' + no) || ''
      : '');

    if (!email && credential && credential.includes('@')) {
      email = credential;
    }

    if (no && (email || credential)) lookup();
  });

  async function lookup() {
    const q = invoiceNo.trim().toUpperCase();
    const em = email.trim().toLowerCase();
    const cred = em || credential.trim();
    if (!q) { error = 'Masukkan nomor invoice.'; return; }
    if (!cred) { error = 'Masukkan email yang digunakan saat pembelian.'; return; }
    if (em && !em.includes('@')) { error = 'Format email tidak valid.'; return; }

    loading = true; error = ''; result = null; requireEmail = false;
    try {
      result = await api.getInvoice(q, cred);
      await syncQrisPreview(result);
    } catch(e) {
      if (e.message.includes('email')) {
        error = 'Email tidak sesuai dengan data pesanan ini. Periksa email yang kamu gunakan saat pembelian.';
      } else if (e.message.includes('tidak ditemukan')) {
        error = 'Invoice tidak ditemukan. Periksa kembali nomor invoice kamu.';
      } else {
        error = e.message;
      }
    } finally { loading = false; }
  }

  function isURL(s) { return s?.startsWith('http'); }

  async function syncQrisPreview(payload) {
    const code = payload?.gateway_qris_string || payload?.gateway_pay_code || '';
    const image = payload?.gateway_qris_image_url || '';
    if (!code || image || typeof window === 'undefined') {
      qrisDataUrl = '';
      return;
    }
    try {
      qrisDataUrl = await toDataURL(code, { margin: 1, scale: 6 });
    } catch {
      qrisDataUrl = '';
    }
  }

  const SC = {
    paid:            { icon:'✅', color:'var(--success-fg)', bg:'var(--success-bg)' },
    pending:         { icon:'⏳', color:'var(--warning-fg)', bg:'var(--warning-bg)' },
    waiting_payment: { icon:'⏳', color:'var(--warning-fg)', bg:'var(--warning-bg)' },
    script_executed: { icon:'⚙️', color:'var(--info-fg)', bg:'var(--info-bg)' },
    expired:         { icon:'⌛', color:'var(--danger-fg)', bg:'var(--danger-bg)' },
    failed:          { icon:'❌', color:'var(--danger-fg)', bg:'var(--danger-bg)' },
    cancelled:       { icon:'✗',  color:'var(--danger-fg)', bg:'var(--danger-bg)' },
  };
  const SC_LABEL = {
    paid:'Lunas', waiting_payment:'Menunggu Pembayaran', verifying:'Pembayaran Diverifikasi', script_executed:'Sedang Diproses',
    expired:'Kadaluarsa', failed:'Gagal', cancelled:'Dibatalkan'
  };
</script>

<svelte:head><title>Cek Invoice — Digital Murah</title></svelte:head>

<nav style="background:var(--surface);border-bottom:0.5px solid var(--border);padding:0 1.5rem;position:sticky;top:0;z-index:100;backdrop-filter:blur(14px)">
  <div style="max-width:700px;margin:0 auto;height:54px;display:flex;align-items:center;gap:10px">
    <a href="/" style="display:flex;align-items:center;gap:8px;font-weight:500;font-size:15px">
      <span style="background:linear-gradient(135deg,var(--primary),var(--primary-2));color:var(--primary-fg);border-radius:8px;width:28px;height:28px;display:flex;align-items:center;justify-content:center;font-size:14px">🛍</span>
      Digital Murah
    </a>
    <ThemeToggle />
    <span style="margin-left:auto;font-size:13px;color:var(--text-muted)">Cek Invoice</span>
  </div>
</nav>

<div style="max-width:680px;margin:0 auto;padding:2rem 1rem">

  <!-- Form cek invoice -->
  <div class="card" style="margin-bottom:1.5rem">
    <div style="font-weight:500;font-size:16px;margin-bottom:4px">🔍 Cek Status Invoice</div>
    <div style="font-size:13px;color:var(--text-muted);margin-bottom:16px">
      Masukkan nomor invoice dan email yang digunakan saat pembelian untuk verifikasi.<br/>
      <span style="font-size:12px">Bisa menggunakan nomor invoice Digital Murah (<code style="font-size:11px">INV-xxx</code>) atau nomor invoice dari SayaBayar/DompetX.</span>
    </div>

    <div style="display:flex;flex-direction:column;gap:10px">
      <div>
        <div style="display:block;font-size:12px;color:var(--text-muted);margin-bottom:4px">Nomor Invoice</div>
        <input class="input mono" placeholder="INV-20260410-123456"
          bind:value={invoiceNo}
          on:keydown={(e) => e.key === 'Enter' && lookup()}
          style="font-size:14px;padding:10px 13px;letter-spacing:0.5px"
        />
      </div>
      <div>
        <div style="display:block;font-size:12px;color:var(--text-muted);margin-bottom:4px">
          Email Pembelian <span style="color:#8c2626">*</span>
        </div>
        <input class="input" type="email" placeholder="email@example.com"
          bind:value={email}
          on:keydown={(e) => e.key === 'Enter' && lookup()}
          style="font-size:14px;padding:10px 13px"
        />
        <div style="font-size:11.5px;color:var(--text-muted);margin-top:4px">
          🔒 Email digunakan untuk memverifikasi kepemilikan invoice. Data tidak disimpan.
        </div>
      </div>
      <button class="btn btn-primary" style="padding:10px;font-size:14px" on:click={lookup} disabled={loading}>
        {loading ? 'Mencari…' : 'Cek Invoice'}
      </button>
    </div>

    {#if error}<div class="alert-error" style="margin-top:10px">{error}</div>{/if}
  </div>

  {#if result}
    {@const sc = SC[statusKey] || SC.waiting_payment}

    <!-- Status banner -->
    <div style="background:{sc.bg};border-radius:var(--radius-lg);padding:1.2rem 1.5rem;margin-bottom:14px;display:flex;align-items:center;gap:14px">
      <span style="font-size:34px">{sc.icon}</span>
      <div>
        <div style="font-weight:500;font-size:16px;color:{sc.color}">{SC_LABEL[statusKey] || statusKey}</div>
        {#if statusKey === 'waiting_payment' && result.expired_at}
          <div style="font-size:12.5px;color:{sc.color};opacity:0.8;margin-top:2px">
            Batas bayar: {fmtDateTime(result.expired_at)}
          </div>
        {:else if statusKey === 'paid'}
          <div style="font-size:12.5px;color:{sc.color};opacity:0.8;margin-top:2px">Produk sudah dikirim ke email Anda</div>
        {/if}
      </div>
      {#if paymentLink}
        <a href={paymentLink} target="_blank" rel="noopener"
          class="btn btn-primary" style="margin-left:auto;white-space:nowrap;font-size:13px">
          💳 Bayar Sekarang
        </a>
      {/if}
    </div>

    {#if statusKey !== 'paid'}
      <div class="card" style="margin-bottom:14px">
        <div style="font-weight:500;font-size:14px;margin-bottom:10px">QRIS Pembayaran</div>
        {#if qrisImage || qrisDataUrl}
          <img src={qrisImage || qrisDataUrl} alt="QRIS" style="width:100%;max-width:280px;margin:0 auto 12px;display:block;border-radius:12px;border:0.5px solid var(--border);background:#fff;padding:8px" />
        {/if}
        {#if qrisString}
          <code style="display:block;background:#f8f8f6;border:0.5px solid var(--border);padding:12px;border-radius:8px;font-size:12px;word-break:break-all;user-select:all">{qrisString}</code>
        {:else if paymentLink}
          <div style="background:#E6F1FB;border-radius:var(--radius);padding:10px 14px;font-size:13px;color:#185FA5;margin-top:0">QRIS belum muncul dari gateway. Gunakan link pembayaran di atas.</div>
        {/if}
      </div>
    {/if}

    <!-- Detail order -->
    <div class="card" style="padding:0;overflow:hidden;margin-bottom:14px">
      <div style="padding:0.9rem 1.25rem;font-weight:500;font-size:14px;border-bottom:0.5px solid var(--border);display:flex;justify-content:space-between;align-items:center">
        <span>Detail Pesanan</span>
        <div style="text-align:right">
          <div class="mono" style="font-size:12px;font-weight:600">{result.invoice_no}</div>
          {#if result.gateway_invoice_no && result.gateway_invoice_no !== result.invoice_no}
            <div style="font-size:11px;color:var(--text-muted)">Gateway: {result.gateway_invoice_no}</div>
          {/if}
        </div>
      </div>
      <table style="width:100%;border-collapse:collapse">
        <tbody>
          {#each [
            ['Produk', result.product_name],
            ['Pembeli', result.buyer_name],
            ['Jumlah', `${result.qty} pcs`],
            ['Metode Bayar', PAY_LABEL[result.pay_method] || result.pay_method],
          ...(result.gateway_provider ? [['Gateway', result.gateway_provider.toUpperCase()]] : []),
        ] as [l, v], i}
            <tr style="background:{i%2===0?'var(--surface-2)':'var(--surface)'}">
              <td style="padding:8px 16px;font-size:12px;color:var(--text-muted);width:40%">{l}</td>
              <td style="padding:8px 16px;font-size:13px;text-align:right">{v}</td>
            </tr>
          {/each}
          <tr style="border-top:2px solid var(--primary)">
            <td style="padding:11px 16px;font-weight:600">Total</td>
            <td style="padding:11px 16px;text-align:right;font-weight:700;font-size:19px;color:var(--primary)">{IDR(result.total)}</td>
          </tr>
        </tbody>
      </table>
    </div>

    <!-- Produk yang diterima -->
    {#if result.status === 'paid' && result.delivered_items?.length}
      <div class="card" style="padding:0;overflow:hidden">
        <div style="padding:0.9rem 1.25rem;font-weight:500;font-size:14px;border-bottom:0.5px solid var(--border);background:#EAF3DE;color:#2f5e0f">
          ✅ Produk Anda
        </div>
        <div style="padding:0.75rem 1rem;display:flex;flex-direction:column;gap:8px">
          {#each result.delivered_items as item, i}
            <div style="display:flex;align-items:flex-start;gap:10px;padding:10px 12px;background:#f8f8f6;border-radius:var(--radius)">
              <div style="width:22px;height:22px;border-radius:50%;background:#0d5fa8;color:#fff;font-size:11px;font-weight:600;display:flex;align-items:center;justify-content:center;flex-shrink:0;margin-top:2px">{i+1}</div>
              {#if isURL(item)}
                <div style="flex:1;min-width:0">
                  <a href={item} target="_blank" rel="noopener"
                    style="display:inline-flex;align-items:center;gap:6px;background:#0d5fa8;color:#fff;padding:7px 16px;border-radius:6px;font-size:13px;font-weight:500;text-decoration:none">
                    📥 Download
                  </a>
                  <div class="mono" style="font-size:10.5px;color:var(--text-muted);margin-top:5px;word-break:break-all">{item}</div>
                </div>
              {:else}
                <div style="flex:1">
                  <div style="font-size:11px;color:var(--text-muted);margin-bottom:4px">Nomer & Pin Untuk Login / Email & Password : </div>
                  <code style="display:block;font-family:'JetBrains Mono',monospace;font-size:13px;background:#fff;border:0.5px solid var(--border);padding:8px 12px;border-radius:6px;word-break:break-all;user-select:all">{item}</code>
                </div>
              {/if}
            </div>
          {/each}
        </div>
      </div>
    {:else if statusKey === 'waiting_payment' || statusKey === 'verifying'}
      <div class="card" style="text-align:center;padding:2rem;color:var(--text-muted)">
        <div style="font-size:28px;margin-bottom:8px">💳</div>
        <div style="font-weight:500;font-size:14px">Menunggu Konfirmasi Pembayaran</div>
        <div style="font-size:13px;margin-top:6px">Produk muncul di sini otomatis setelah pembayaran dikonfirmasi.</div>
      </div>
    {/if}

    <div style="margin-top:14px;text-align:center">
      <button class="btn btn-sm" on:click={()=>{result=null;error=''}}>← Cek Invoice Lain</button>
    </div>

    <div style="margin-top:10px;text-align:center">
        <a href="/komplain?invoice={result.invoice_no}" style="font-size:12.5px;color:var(--primary);font-weight:500">
        Komplain ke WhatsApp Admin →
      </a>
    </div>
  {/if}
</div>

<script>
  import { onMount } from 'svelte';
  import { page } from '$app/stores';
  import { api } from '$lib/api.js';
  import { IDR, fmtDateTime, STATUS_LABEL, PAY_LABEL } from '$lib/utils.js';

  let invoiceNo = '';
  let email = '';
  let result = null;
  let loading = false;
  let error = '';
  let requireEmail = false;

  onMount(() => {
    const no = $page.url.searchParams.get('no');
    const em = $page.url.searchParams.get('email');
    if (no) invoiceNo = no;
    if (em) email = em;
    if (no && em) lookup();
  });

  async function lookup() {
    const q = invoiceNo.trim().toUpperCase();
    const em = email.trim().toLowerCase();
    if (!q) { error = 'Masukkan nomor invoice.'; return; }
    if (!em) { error = 'Masukkan email yang digunakan saat pembelian.'; return; }
    if (!em.includes('@')) { error = 'Format email tidak valid.'; return; }

    loading = true; error = ''; result = null; requireEmail = false;
    try {
      result = await api.getInvoice(q, em);
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

  const SC = {
    paid:            { icon:'✅', color:'#2f5e0f', bg:'#EAF3DE' },
    pending:         { icon:'⏳', color:'#854F0B', bg:'#FAEEDA' },
    script_executed: { icon:'⚙️', color:'#185FA5', bg:'#E6F1FB' },
    expired:         { icon:'⌛', color:'#8c2626', bg:'#FCEBEB' },
    failed:          { icon:'❌', color:'#8c2626', bg:'#FCEBEB' },
    cancelled:       { icon:'✗',  color:'#8c2626', bg:'#FCEBEB' },
  };
  const SC_LABEL = {
    paid:'Lunas', pending:'Menunggu Pembayaran', script_executed:'Sedang Diproses',
    expired:'Kadaluarsa', failed:'Gagal', cancelled:'Dibatalkan'
  };
</script>

<svelte:head><title>Cek Invoice — DigiStore</title></svelte:head>

<nav style="background:#fff;border-bottom:0.5px solid var(--border);padding:0 1.5rem">
  <div style="max-width:700px;margin:0 auto;height:54px;display:flex;align-items:center;gap:10px">
    <a href="/" style="display:flex;align-items:center;gap:8px;font-weight:500;font-size:15px">
      <span style="background:#0d5fa8;border-radius:8px;width:28px;height:28px;display:flex;align-items:center;justify-content:center;font-size:14px">🛍</span>
      DigiStore
    </a>
    <span style="margin-left:auto;font-size:13px;color:var(--text-muted)">Cek Invoice</span>
  </div>
</nav>

<div style="max-width:680px;margin:0 auto;padding:2rem 1rem">

  <!-- Form cek invoice -->
  <div class="card" style="margin-bottom:1.5rem">
    <div style="font-weight:500;font-size:16px;margin-bottom:4px">🔍 Cek Status Invoice</div>
    <div style="font-size:13px;color:var(--text-muted);margin-bottom:16px">
      Masukkan nomor invoice dan email yang digunakan saat pembelian untuk verifikasi.<br/>
      <span style="font-size:12px">Bisa menggunakan nomor invoice DigiStore (<code style="font-size:11px">INV-xxx</code>) atau nomor invoice dari SayaBayar/DompetX.</span>
    </div>

    <div style="display:flex;flex-direction:column;gap:10px">
      <div>
        <label style="display:block;font-size:12px;color:var(--text-muted);margin-bottom:4px">Nomor Invoice</label>
        <input class="input mono" placeholder="INV-20260410-123456"
          bind:value={invoiceNo}
          on:keydown={(e) => e.key === 'Enter' && lookup()}
          style="font-size:14px;padding:10px 13px;letter-spacing:0.5px"
        />
      </div>
      <div>
        <label style="display:block;font-size:12px;color:var(--text-muted);margin-bottom:4px">
          Email Pembelian <span style="color:#8c2626">*</span>
        </label>
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
    {@const sc = SC[result.status] || SC.pending}

    <!-- Status banner -->
    <div style="background:{sc.bg};border-radius:var(--radius-lg);padding:1.2rem 1.5rem;margin-bottom:14px;display:flex;align-items:center;gap:14px">
      <span style="font-size:34px">{sc.icon}</span>
      <div>
        <div style="font-weight:500;font-size:16px;color:{sc.color}">{SC_LABEL[result.status] || result.status}</div>
        {#if result.status === 'pending' && result.expired_at}
          <div style="font-size:12.5px;color:{sc.color};opacity:0.8;margin-top:2px">
            Batas bayar: {fmtDateTime(result.expired_at)}
          </div>
        {:else if result.status === 'paid'}
          <div style="font-size:12.5px;color:{sc.color};opacity:0.8;margin-top:2px">Produk sudah dikirim ke email Anda</div>
        {/if}
      </div>
      {#if result.status === 'pending' && result.gateway_pay_url}
        <a href={result.gateway_pay_url} target="_blank" rel="noopener"
          class="btn btn-primary" style="margin-left:auto;white-space:nowrap;font-size:13px">
          💳 Bayar Sekarang
        </a>
      {/if}
    </div>

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
        {#each [
          ['Produk', result.product_name],
          ['Pembeli', result.buyer_name],
          ['Jumlah', `${result.qty} pcs`],
          ['Metode Bayar', PAY_LABEL[result.pay_method] || result.pay_method],
          ...(result.gateway_provider ? [['Gateway', result.gateway_provider.toUpperCase()]] : []),
        ] as [l, v], i}
          <tr style="background:{i%2===0?'#f9f9f9':'#fff'}">
            <td style="padding:8px 16px;font-size:12px;color:var(--text-muted);width:40%">{l}</td>
            <td style="padding:8px 16px;font-size:13px;text-align:right">{v}</td>
          </tr>
        {/each}
        <tbody>
          <tr style="border-top:2px solid #0d5fa8">
          <td style="padding:11px 16px;font-weight:600">Total</td>
          <td style="padding:11px 16px;text-align:right;font-weight:700;font-size:19px;color:#0d5fa8">{IDR(result.total)}</td>
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
                  <div style="font-size:11px;color:var(--text-muted);margin-bottom:4px">License Key:</div>
                  <code style="display:block;font-family:'JetBrains Mono',monospace;font-size:13px;background:#fff;border:0.5px solid var(--border);padding:8px 12px;border-radius:6px;word-break:break-all;user-select:all">{item}</code>
                </div>
              {/if}
            </div>
          {/each}
        </div>
      </div>
    {:else if result.status === 'pending'}
      <div class="card" style="text-align:center;padding:2rem;color:var(--text-muted)">
        <div style="font-size:28px;margin-bottom:8px">💳</div>
        <div style="font-weight:500;font-size:14px">Menunggu Konfirmasi Pembayaran</div>
        <div style="font-size:13px;margin-top:6px">Produk muncul di sini otomatis setelah pembayaran dikonfirmasi.</div>
      </div>
    {/if}

    <div style="margin-top:14px;text-align:center">
      <button class="btn btn-sm" on:click={()=>{result=null;error=''}}>← Cek Invoice Lain</button>
    </div>
  {/if}
</div>

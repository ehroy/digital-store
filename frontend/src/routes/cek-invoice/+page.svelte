<script>
  import { onMount } from 'svelte';
  import { page } from '$app/stores';
  import { api } from '$lib/api.js';
  import { IDR, fmtDateTime, STATUS_LABEL, PAY_LABEL } from '$lib/utils.js';

  let invoiceNo = '';
  let result = null;
  let loading = false;
  let error = '';

  onMount(() => {
    const no = $page.url.searchParams.get('no');
    if (no) { invoiceNo = no; lookup(); }
  });

  async function lookup() {
    const q = invoiceNo.trim().toUpperCase();
    if (!q) { error = 'Masukkan nomor invoice terlebih dahulu.'; return; }
    loading = true; error = ''; result = null;
    try {
      result = await api.getInvoice(q);
    } catch(e) {
      error = e.message === 'invoice tidak ditemukan'
        ? 'Invoice tidak ditemukan. Periksa kembali nomor invoice Anda.'
        : e.message;
    } finally { loading = false; }
  }

  function isURL(s) { return s?.startsWith('http://') || s?.startsWith('https://'); }

  const statusConfig = {
    paid:            { color:'#2f5e0f', bg:'#EAF3DE', icon:'✅', label:'Lunas' },
    pending:         { color:'#854F0B', bg:'#FAEEDA', icon:'⏳', label:'Menunggu Pembayaran' },
    script_executed: { color:'#185FA5', bg:'#E6F1FB', icon:'⚙️', label:'Sedang Diproses' },
    cancelled:       { color:'#8c2626', bg:'#FCEBEB', icon:'✗',  label:'Dibatalkan' },
  };
</script>

<svelte:head><title>Cek Invoice — Digitalku Murah</title></svelte:head>

<nav style="background:#fff;border-bottom:0.5px solid var(--border);padding:0 1.5rem">
  <div style="max-width:720px;margin:0 auto;height:54px;display:flex;align-items:center;gap:10px">
    <a href="/" style="display:flex;align-items:center;gap:8px;font-weight:500;font-size:15px">
      <span style="background:#0d5fa8;border-radius:8px;width:28px;height:28px;display:flex;align-items:center;justify-content:center;font-size:14px">🛍</span>
      Digitalku Murah
    </a>
    <span style="margin-left:auto;font-size:13px;color:var(--text-muted)">Cek Status Invoice</span>
  </div>
</nav>

<div style="max-width:680px;margin:0 auto;padding:2rem 1rem">

  <!-- Search box -->
  <div class="card" style="margin-bottom:1.5rem">
    <div style="font-weight:500;font-size:16px;margin-bottom:4px">🔍 Cek Status Invoice</div>
    <div style="font-size:13px;color:var(--text-muted);margin-bottom:16px">
      Masukkan nomor invoice yang kamu terima di email untuk melihat status pembayaran dan produk yang dikirim.
    </div>
    <div style="display:flex;gap:8px">
      <input
        class="input mono"
        placeholder="INV-20260410-123456"
        bind:value={invoiceNo}
        on:keydown={(e) => e.key === 'Enter' && lookup()}
        style="flex:1;font-size:15px;padding:11px 14px"
      />
      <button class="btn btn-primary" style="padding:11px 20px;font-size:14px" on:click={lookup} disabled={loading}>
        {loading ? 'Mencari…' : 'Cek'}
      </button>
    </div>
    {#if error}<div class="alert-error" style="margin-top:10px">{error}</div>{/if}
  </div>

  <!-- Result -->
  {#if result}
    {@const sc = statusConfig[result.status] || statusConfig.pending}

    <!-- Status banner -->
    <div style="background:{sc.bg};border-radius:var(--radius-lg);padding:1.25rem 1.5rem;margin-bottom:14px;display:flex;align-items:center;gap:14px">
      <span style="font-size:36px">{sc.icon}</span>
      <div>
        <div style="font-weight:500;font-size:16px;color:{sc.color}">{sc.label}</div>
        <div style="font-size:13px;color:{sc.color};opacity:0.8;margin-top:2px">
          {#if result.status === 'paid'}
            Pembayaran dikonfirmasi. Produk telah dikirim ke email Anda.
          {:else if result.status === 'pending'}
            Kami belum menerima konfirmasi pembayaran. Harap selesaikan pembayaran Anda.
          {:else if result.status === 'script_executed'}
            Tim kami sedang memproses pesanan Anda. Akan dihubungi dalam 1×24 jam kerja.
          {:else}
            Pesanan ini telah dibatalkan.
          {/if}
        </div>
      </div>
    </div>

    <!-- Invoice detail -->
    <div class="card" style="padding:0;overflow:hidden;margin-bottom:14px">
      <div style="padding:1rem 1.25rem;border-bottom:0.5px solid var(--border);display:flex;justify-content:space-between;align-items:center">
        <div>
          <div style="font-weight:500;font-size:15px">Digitalku Murah</div>
          <div style="font-size:12px;color:var(--text-muted)">Produk Digital Indonesia</div>
        </div>
        <div style="text-align:right">
          <div class="mono" style="font-size:13px;font-weight:600">{result.invoice_no}</div>
          <div style="font-size:12px;color:var(--text-muted)">{fmtDateTime(result.created_at)}</div>
        </div>
      </div>
      <table style="width:100%;border-collapse:collapse">
        {#each [
          ['Produk', result.product_name],
          ['Pembeli', result.buyer_name],
          ['Jumlah', `${result.qty} pcs`],
          ['Metode Bayar', PAY_LABEL[result.pay_method] || result.pay_method],
          ['Tipe', result.product_type === 'stock' ? 'Produk Digital (Stok)' : 'Layanan / Jasa'],
        ] as [l, v], i}
          <tbody>
            <tr style="background:{i%2===0?'#f9f9f9':'#fff'}">
              <td style="padding:9px 16px;font-size:12px;color:var(--text-muted);width:40%">{l}</td>
              <td style="padding:9px 16px;font-size:13px;text-align:right">{v}</td>
            </tr>
          </tbody>
        {/each}
        <tbody>
          <tr style="border-top:2px solid #0d5fa8">
            <td style="padding:12px 16px;font-weight:600;font-size:15px">TOTAL</td>
            <td style="padding:12px 16px;text-align:right;font-weight:700;font-size:20px;color:#0d5fa8">{IDR(result.total)}</td>
          </tr>
        </tbody>
      </table>
    </div>

    <!-- Delivered items (hanya jika paid dan ada item) -->
    {#if result.status === 'paid' && result.delivered_items?.length}
      <div class="card" style="padding:0;overflow:hidden">
        <div style="padding:1rem 1.25rem;border-bottom:0.5px solid var(--border);font-weight:500;font-size:14px">
          📦 Produk yang Dikirim
        </div>
        <div style="padding:0.75rem 1rem;display:flex;flex-direction:column;gap:8px">
          {#each result.delivered_items as item, i}
            <div style="display:flex;align-items:center;gap:10px;padding:10px 12px;background:#f8f8f6;border-radius:var(--radius)">
              <div style="width:22px;height:22px;border-radius:50%;background:#0d5fa8;color:#fff;font-size:12px;font-weight:500;display:flex;align-items:center;justify-content:center;flex-shrink:0">{i+1}</div>
              {#if isURL(item)}
                <div style="flex:1;min-width:0">
                  <a href={item} target="_blank" rel="noopener"
                    style="display:inline-flex;align-items:center;gap:6px;background:#0d5fa8;color:#fff;padding:6px 14px;border-radius:6px;font-size:13px;font-weight:500;text-decoration:none">
                    📥 Download →
                  </a>
                  <div class="mono" style="font-size:11px;color:var(--text-muted);margin-top:4px;word-break:break-all">{item}</div>
                </div>
              {:else}
                <div style="flex:1">
                  <div style="font-size:12px;color:var(--text-muted);margin-bottom:3px">License / Key:</div>
                  <code style="font-family:'JetBrains Mono',monospace;font-size:13px;background:#fff;border:0.5px solid var(--border);padding:6px 12px;border-radius:6px;display:block;word-break:break-all;user-select:all">{item}</code>
                </div>
              {/if}
            </div>
          {/each}
        </div>
        <div style="padding:0.75rem 1.25rem;background:#f8f8f6;border-top:0.5px solid var(--border);font-size:12px;color:var(--text-muted)">
          ⚠️ Simpan halaman ini atau email invoice Anda. Item berlaku permanen.
        </div>
      </div>

    {:else if result.status === 'pending'}
      <div class="card" style="text-align:center;padding:2rem;color:var(--text-muted)">
        <div style="font-size:32px;margin-bottom:8px">💳</div>
        <div style="font-size:14px;font-weight:500">Menunggu Pembayaran</div>
        <div style="font-size:13px;margin-top:6px">Setelah pembayaran dikonfirmasi, produk akan otomatis muncul di halaman ini.</div>
      </div>

    {:else if result.status === 'script_executed'}
      <div class="card" style="text-align:center;padding:2rem;color:var(--text-muted)">
        <div style="font-size:32px;margin-bottom:8px">👷</div>
        <div style="font-size:14px;font-weight:500">Pesanan Sedang Diproses</div>
        <div style="font-size:13px;margin-top:6px">Tim kami akan menghubungi Anda melalui email dalam 1×24 jam kerja.</div>
      </div>
    {/if}

    <div style="margin-top:16px;text-align:center">
      <a href="/" style="font-size:13px;color:var(--text-muted)">← Kembali ke Toko</a>
    </div>
  {/if}
</div>

<script>
  import { onMount } from 'svelte';
  import { page } from '$app/stores';
  import { api } from '$lib/api.js';
  import { IDR } from '$lib/utils.js';

  const invoiceNo = $page.url.searchParams.get('invoice') || '';

  let contact = null;
  let loading = true;

  // Form state
  let name = '';
  let email = '';
  let invoiceInput = invoiceNo;
  let issue = '';
  let phone = '';
  let invoiceData = null;
  let emailVerified = false;
  let verifying = false;
  let verifyError = '';

  onMount(async () => {
    try { contact = await api.getContact(); } catch {}
    finally { loading = false; }
  });

  async function verifyInvoice() {
    if (!invoiceInput.trim()) { verifyError = 'Nomor invoice wajib diisi.'; return; }
    if (!email.trim() || !email.includes('@')) { verifyError = 'Email tidak valid.'; return; }
    verifying = true; verifyError = ''; invoiceData = null; emailVerified = false;
    try {
      invoiceData = await api.getInvoice(invoiceInput.trim().toUpperCase(), email.trim());
      name = invoiceData.buyer_name || name;
      emailVerified = true;
    } catch(e) {
      verifyError = e.message?.includes('email') || e.message?.includes('sesuai')
        ? 'Email tidak sesuai dengan invoice ini. Gunakan email yang dipakai saat pembelian.'
        : e.message?.includes('tidak ditemukan')
        ? 'Nomor invoice tidak ditemukan.'
        : (e.message || 'Terjadi kesalahan.');
    } finally { verifying = false; }
  }

  function renderTemplate() {
    const tmpl = contact?.complaint_template ||
      'Halo admin, saya ingin komplain order berikut:\n\nInvoice: {invoice_no}\nProduk: {product_name}\nNama: {buyer_name}\nEmail: {buyer_email}\nNomor HP: {phone}\nStatus: {status}\n\nMasalah:\n{issue}';
    const statusLabel = invoiceData?.status === 'paid' ? 'Lunas' : (invoiceData?.status || '-');
    return tmpl
      .replaceAll('{invoice_no}', invoiceData?.invoice_no || invoiceInput || '-')
      .replaceAll('{product_name}', invoiceData?.product_name || '-')
      .replaceAll('{buyer_name}', name || '-')
      .replaceAll('{buyer_email}', email || '-')
      .replaceAll('{phone}', phone || '-')
      .replaceAll('{status}', statusLabel)
      .replaceAll('{issue}', issue || '[belum diisi]');
  }

  function openWA() {
    const no = (contact?.whatsapp || '').replace(/\D/g, '');
    if (!no) return;
    window.open(`https://wa.me/${no}?text=${encodeURIComponent(renderTemplate())}`, '_blank', 'noopener');
  }

  $: waOk = !!(contact?.whatsapp);
  $: canSend = issue.trim().length >= 10 && emailVerified;
</script>

<svelte:head><title>Komplain & Bantuan — DigiStore</title></svelte:head>

<nav style="background:#fff;border-bottom:0.5px solid var(--border);padding:0 1.5rem">
  <div style="max-width:700px;margin:0 auto;height:54px;display:flex;align-items:center;gap:10px">
    <a href="/" style="display:flex;align-items:center;gap:8px;font-weight:500;font-size:15px">
      <span style="background:#0d5fa8;border-radius:8px;width:28px;height:28px;display:flex;align-items:center;justify-content:center;font-size:14px">🛍</span>
      {contact?.business_name || 'DigiStore'}
    </a>
    <span style="margin-left:auto;font-size:13px;color:var(--text-muted)">Komplain & Bantuan</span>
  </div>
</nav>

<div style="max-width:680px;margin:0 auto;padding:2rem 1rem 4rem">

  <!-- Header -->
  <div style="text-align:center;margin-bottom:2rem">
    <div style="font-size:40px;margin-bottom:10px">🎧</div>
    <h1 style="font-size:22px;font-weight:500;margin-bottom:6px">Pusat Bantuan</h1>
    <p style="font-size:13.5px;color:var(--text-muted)">
      Ada masalah dengan pesanan? Kami siap membantu.
    </p>
    {#if contact?.operational_hours}
      <p style="font-size:12.5px;color:var(--text-muted);margin-top:4px">⏰ {contact.operational_hours}</p>
    {/if}
  </div>

  <!-- Form -->
  <div class="card" style="margin-bottom:16px">
    <div style="font-weight:500;font-size:15px;margin-bottom:16px">📝 Form Komplain</div>

    <!-- Step 1 -->
    <div class="step-wrap {emailVerified?'verified':''}">
      <div class="step-num">1</div>
      <div style="flex:1">
        <div style="font-weight:500;font-size:13.5px;margin-bottom:10px">
          Verifikasi Pesanan {emailVerified?'✓':''}
        </div>
        {#if emailVerified && invoiceData}
          <div style="background:#EAF3DE;border-radius:var(--radius);padding:10px 13px;font-size:13px;color:#2f5e0f">
            ✓ <strong>{invoiceData.invoice_no}</strong> — {invoiceData.product_name} — {IDR(invoiceData.total)}
          </div>
          <button class="btn btn-sm" style="margin-top:8px;font-size:12px"
            on:click={() => { emailVerified = false; invoiceData = null; }}>Ganti Invoice</button>
        {:else}
          <div class="form-row-2" style="gap:10px;margin-bottom:10px">
            <div>
              <label class="field-label">Nomor Invoice *</label>
              <input class="input mono" bind:value={invoiceInput}
                placeholder="INV-20260410-123456"
                on:keydown={(e) => e.key === 'Enter' && verifyInvoice()} />
            </div>
            <div>
              <label class="field-label">Email Pembelian *</label>
              <input class="input" type="email" bind:value={email}
                placeholder="email@example.com"
                on:keydown={(e) => e.key === 'Enter' && verifyInvoice()} />
            </div>
          </div>
          {#if verifyError}
            <div class="alert-error" style="margin-bottom:10px">{verifyError}</div>
          {/if}
          <button class="btn btn-primary" style="padding:8px 20px"
            on:click={verifyInvoice} disabled={verifying}>
            {verifying ? 'Memeriksa…' : 'Verifikasi Invoice'}
          </button>
        {/if}
      </div>
    </div>

    <!-- Step 2: Nama + HP -->
    <div class="step-wrap" class:disabled={!emailVerified}>
      <div class="step-num">2</div>
      <div style="flex:1">
        <div style="font-weight:500;font-size:13.5px;margin-bottom:8px">Nama Kamu</div>
        <input class="input" bind:value={name} placeholder="Nama lengkap" disabled={!emailVerified} />
        <input class="input" style="margin-top:8px" bind:value={phone} placeholder="Nomor WhatsApp aktif (opsional)" disabled={!emailVerified} />
      </div>
    </div>

    <!-- Step 3: Masalah -->
    <div class="step-wrap" class:disabled={!emailVerified}>
      <div class="step-num">3</div>
      <div style="flex:1">
        <div style="font-weight:500;font-size:13.5px;margin-bottom:8px">Jelaskan Masalah *</div>
        <textarea class="input" rows="4" style="resize:vertical"
          bind:value={issue} disabled={!emailVerified}
          placeholder="Contoh: Link yang saya terima tidak bisa dibuka / item belum masuk setelah bayar...">
        </textarea>
        <div style="font-size:11.5px;color:var(--text-muted);margin-top:4px">
          {issue.length} karakter (minimal 10)
        </div>
      </div>
    </div>

    <!-- Step 4: Kirim -->
    <div class="step-wrap" class:disabled={!canSend || !emailVerified}>
      <div class="step-num">4</div>
      <div style="flex:1">
        <div style="font-weight:500;font-size:13.5px;margin-bottom:10px">Kirim Komplain</div>

        {#if !waOk}
          <div style="font-size:13px;color:var(--text-muted);padding:12px;background:#f8f8f6;border-radius:var(--radius)">
            Admin belum mengatur nomor WhatsApp.
          </div>
        {:else}
          <button class="btn btn-primary" style="width:100%;padding:12px 16px"
            on:click={openWA}
            disabled={!canSend}>
            Lanjut ke WhatsApp Admin
          </button>
          <p style="font-size:12px;color:var(--text-muted);margin-top:8px">
            Pesan akan terisi otomatis sesuai template admin dan data invoice.
          </p>
        {/if}
      </div>
    </div>
  </div>

  <!-- Info kontak -->
  {#if contact && (contact.email || contact.instagram || contact.telegram)}
    <div class="card">
      <div style="font-weight:500;font-size:14px;margin-bottom:12px">📬 Kontak Lainnya</div>
      <div style="display:flex;flex-direction:column;gap:8px">
        {#if contact.email}
          <div style="display:flex;align-items:center;gap:8px;font-size:13.5px">
            ✉️ <a href="mailto:{contact.email}" style="color:#0d5fa8">{contact.email}</a>
          </div>
        {/if}
        {#if contact.instagram}
          <div style="display:flex;align-items:center;gap:8px;font-size:13.5px">
            📸 <a href="https://instagram.com/{contact.instagram.replace('@','')}" target="_blank" style="color:#0d5fa8">{contact.instagram}</a>
          </div>
        {/if}
        {#if contact.telegram}
          <div style="display:flex;align-items:center;gap:8px;font-size:13.5px">
            💬 <a href="https://t.me/{contact.telegram.replace('@','')}" target="_blank" style="color:#0d5fa8">{contact.telegram}</a>
          </div>
        {/if}
        {#if contact.website}
          <div style="display:flex;align-items:center;gap:8px;font-size:13.5px">
            🌐 <a href={contact.website} target="_blank" style="color:#0d5fa8">{contact.website}</a>
          </div>
        {/if}
      </div>
    </div>
  {/if}
</div>

<style>
.step-wrap {
  display: flex; align-items: flex-start; gap: 14px;
  padding: 14px 0; border-bottom: 0.5px solid var(--border);
  transition: opacity 0.2s;
}
.step-wrap:last-child { border-bottom: none; }
.step-wrap.disabled { opacity: 0.4; pointer-events: none; }
.step-wrap.verified .step-num { background: #2f5e0f; }
.step-num {
  width: 26px; height: 26px; border-radius: 50%;
  background: #0d5fa8; color: #fff;
  font-size: 13px; font-weight: 600;
  display: flex; align-items: center; justify-content: center;
  flex-shrink: 0; margin-top: 2px;
}

</style>

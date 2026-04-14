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
      'Halo, saya ingin mengajukan komplain:\n\nNomor Invoice: {invoice_no}\nProduk: {product_name}\nNama: {buyer_name}\n\nMasalah:\n{issue}';
    return tmpl
      .replace('{invoice_no}', invoiceData?.invoice_no || invoiceInput || '-')
      .replace('{product_name}', invoiceData?.product_name || '-')
      .replace('{buyer_name}', name || '-')
      .replace('{issue}', issue || '[belum diisi]');
  }

  function openWA() {
    const no = (contact?.whatsapp || '').replace(/\D/g, '');
    if (!no) return;
    window.open(`https://wa.me/${no}?text=${encodeURIComponent(renderTemplate())}`, '_blank', 'noopener');
  }

  function openTG() {
    const tg = (contact?.telegram || '').replace('@', '').replace('https://t.me/', '');
    if (!tg) return;
    window.open(`https://t.me/${tg}?text=${encodeURIComponent(renderTemplate())}`, '_blank', 'noopener');
  }

  $: waOk = !!(contact?.whatsapp);
  $: tgOk = !!(contact?.telegram);
  $: canSend = issue.trim().length >= 10;
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

    <!-- Step 2: Nama -->
    <div class="step-wrap" class:disabled={!emailVerified}>
      <div class="step-num">2</div>
      <div style="flex:1">
        <div style="font-weight:500;font-size:13.5px;margin-bottom:8px">Nama Kamu</div>
        <input class="input" bind:value={name} placeholder="Nama lengkap" disabled={!emailVerified} />
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

        {#if !waOk && !tgOk && !contact?.email}
          <div style="font-size:13px;color:var(--text-muted);padding:12px;background:#f8f8f6;border-radius:var(--radius)">
            Admin belum mengatur informasi kontak. Hubungi kami melalui media sosial.
          </div>
        {:else}
          <div style="display:flex;gap:10px;flex-wrap:wrap">
            {#if waOk}
              <button class="btn-channel wa"
                on:click={openWA}
                disabled={!canSend || !emailVerified}>
                <svg viewBox="0 0 24 24" width="20" height="20" fill="currentColor">
                  <path d="M17.472 14.382c-.297-.149-1.758-.867-2.03-.967-.273-.099-.471-.148-.67.15-.197.297-.767.966-.94 1.164-.173.199-.347.223-.644.075-.297-.15-1.255-.463-2.39-1.475-.883-.788-1.48-1.761-1.653-2.059-.173-.297-.018-.458.13-.606.134-.133.298-.347.446-.52.149-.174.198-.298.298-.497.099-.198.05-.371-.025-.52-.075-.149-.669-1.612-.916-2.207-.242-.579-.487-.5-.669-.51-.173-.008-.371-.01-.57-.01-.198 0-.52.074-.792.372-.272.297-1.04 1.016-1.04 2.479 0 1.462 1.065 2.875 1.213 3.074.149.198 2.096 3.2 5.077 4.487.709.306 1.262.489 1.694.625.712.227 1.36.195 1.871.118.571-.085 1.758-.719 2.006-1.413.248-.694.248-1.289.173-1.413-.074-.124-.272-.198-.57-.347m-5.421 7.403h-.004a9.87 9.87 0 01-5.031-1.378l-.361-.214-3.741.982.998-3.648-.235-.374a9.86 9.86 0 01-1.51-5.26c.001-5.45 4.436-9.884 9.888-9.884 2.64 0 5.122 1.03 6.988 2.898a9.825 9.825 0 012.893 6.994c-.003 5.45-4.437 9.884-9.885 9.884m8.413-18.297A11.815 11.815 0 0012.05 0C5.495 0 .16 5.335.157 11.892c0 2.096.547 4.142 1.588 5.945L.057 24l6.305-1.654a11.882 11.882 0 005.683 1.448h.005c6.554 0 11.89-5.335 11.893-11.893a11.821 11.821 0 00-3.48-8.413z"/>
                </svg>
                WhatsApp
              </button>
            {/if}

            {#if tgOk}
              <button class="btn-channel tg"
                on:click={openTG}
                disabled={!canSend || !emailVerified}>
                <svg viewBox="0 0 24 24" width="20" height="20" fill="currentColor">
                  <path d="M11.944 0A12 12 0 0 0 0 12a12 12 0 0 0 12 12 12 12 0 0 0 12-12A12 12 0 0 0 12 0a12 12 0 0 0-.056 0zm4.962 7.224c.1-.002.321.023.465.14a.506.506 0 0 1 .171.325c.016.093.036.306.02.472-.18 1.898-.962 6.502-1.36 8.627-.168.9-.499 1.201-.82 1.23-.696.065-1.225-.46-1.9-.902-1.056-.693-1.653-1.124-2.678-1.8-1.185-.78-.417-1.21.258-1.91.177-.184 3.247-2.977 3.307-3.23.007-.032.014-.15-.056-.212s-.174-.041-.249-.024c-.106.024-1.793 1.14-5.061 3.345-.48.33-.913.49-1.302.48-.428-.008-1.252-.241-1.865-.44-.752-.245-1.349-.374-1.297-.789.027-.216.325-.437.893-.663 3.498-1.524 5.83-2.529 6.998-3.014 3.332-1.386 4.025-1.627 4.476-1.635z"/>
                </svg>
                Telegram
              </button>
            {/if}

            {#if contact?.email}
              <a href="mailto:{contact.email}?subject=Komplain {invoiceInput}&body={encodeURIComponent(renderTemplate())}"
                class="btn-channel email" style="text-decoration:none">
                ✉️ Email
              </a>
            {/if}
          </div>

          {#if !canSend || !emailVerified}
            <p style="font-size:12px;color:var(--text-muted);margin-top:8px">
              {!emailVerified ? '⚠️ Verifikasi invoice terlebih dahulu' : '⚠️ Masukkan deskripsi masalah (minimal 10 karakter)'}
            </p>
          {/if}
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

.btn-channel {
  display: inline-flex; align-items: center; gap: 8px;
  padding: 11px 20px; border-radius: var(--radius);
  border: none; cursor: pointer;
  font-size: 14px; font-weight: 500;
  font-family: inherit;
  transition: opacity 0.15s, transform 0.15s;
}
.btn-channel:not(:disabled):hover { opacity: 0.88; transform: translateY(-1px); }
.btn-channel:disabled { opacity: 0.35; cursor: not-allowed; }
.btn-channel.wa    { background: #25D366; color: #fff; }
.btn-channel.tg    { background: #229ED9; color: #fff; }
.btn-channel.email { background: #0d5fa8; color: #fff; }
</style>

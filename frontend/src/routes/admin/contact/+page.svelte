<script>
  import { onMount } from 'svelte';
  import { api } from '$lib/api.js';

  let cfg = null;
  let loading = true;
  let saving = false;
  let saved = false;
  let error = '';

  onMount(async () => {
    try { cfg = await api.getAdminContact(); }
    catch(e) { error = e.message; }
    finally { loading = false; }
  });

  async function save() {
    saving = true; error = ''; saved = false;
    try {
      cfg = await api.updateContact(cfg);
      saved = true; setTimeout(() => saved = false, 2500);
    } catch(e) { error = e.message; }
    finally { saving = false; }
  }

  // Preview link WA
  $: waLink = cfg?.whatsapp
    ? `https://wa.me/${cfg.whatsapp.replace(/\D/g,'')}`
    : '';
  $: tgLink = cfg?.telegram
    ? (cfg.telegram.startsWith('http') ? cfg.telegram : `https://t.me/${cfg.telegram.replace('@','')}`)
    : '';
</script>

<svelte:head><title>Kontak & Support — Digital Murah Admin</title></svelte:head>

<div class="page-header"><h1 class="page-title">📞 Kontak & Support</h1></div>
<p style="font-size:13px;color:var(--text-muted);margin-bottom:1.25rem">
  Konfigurasi tombol WhatsApp, media sosial, dan template pesan komplain yang tampil di toko.
</p>

{#if loading}
  <div style="color:var(--text-muted);padding:2rem">Memuat…</div>
{:else if cfg}
<div style="display:flex;flex-direction:column;gap:14px">

  <!-- Identitas bisnis -->
  <div class="card">
    <div style="font-weight:500;font-size:14px;margin-bottom:14px">🏪 Identitas Bisnis</div>
    <div class="form-row-2">
      <div>
        <label class="field-label">Nama Bisnis</label>
        <input class="input" bind:value={cfg.business_name} placeholder="Digital Murah" />
      </div>
      <div>
        <label class="field-label">Website</label>
        <input class="input" bind:value={cfg.website} placeholder="https://yourdomain.com" />
      </div>
    </div>
    <div style="margin-top:12px">
      <label class="field-label">Deskripsi Bisnis</label>
      <textarea class="input" rows="2" style="resize:vertical" bind:value={cfg.business_desc}
        placeholder="Toko digital terpercaya, produk instan 24 jam."></textarea>
    </div>
  </div>

  <!-- WhatsApp -->
  <div class="card">
    <div style="font-weight:500;font-size:14px;margin-bottom:14px">
      <span style="color:#25D366">●</span> WhatsApp
    </div>
    <div class="form-row-2">
      <div>
        <label class="field-label">Nomor WhatsApp</label>
        <input class="input" bind:value={cfg.whatsapp}
          placeholder="6281234567890 (dengan kode negara, tanpa +)" />
        <div style="font-size:11.5px;color:var(--text-muted);margin-top:3px">
          Format: 628xxx (62 = Indonesia). Contoh: 6281234567890
        </div>
      </div>
      <div>
        <label class="field-label">Label Tombol</label>
        <input class="input" bind:value={cfg.whatsapp_label} placeholder="Hubungi CS" />
      </div>
    </div>
    {#if waLink}
      <div style="margin-top:10px;padding:8px 12px;background:#f0fff4;border-radius:var(--radius);font-size:12.5px;color:#2f5e0f">
        Preview link: <a href={waLink} target="_blank" style="color:#2f5e0f;font-weight:500">{waLink}</a>
      </div>
    {/if}
  </div>

  <!-- Sosial media lain -->
  <div class="card">
    <div style="font-weight:500;font-size:14px;margin-bottom:14px">📱 Media Sosial & Kontak Lain</div>
    <div class="form-row-2">
      <div>
        <label class="field-label">Telegram (username atau link)</label>
        <input class="input" bind:value={cfg.telegram} placeholder="@username atau https://t.me/xxx" />
        {#if tgLink}
          <a href={tgLink} target="_blank" style="font-size:11.5px;color:#0d5fa8;margin-top:3px;display:block">{tgLink}</a>
        {/if}
      </div>
      <div>
        <label class="field-label">Instagram</label>
        <input class="input" bind:value={cfg.instagram} placeholder="@username" />
      </div>
    </div>
    <div style="margin-top:12px">
      <label class="field-label">Email Support</label>
      <input class="input" type="email" bind:value={cfg.email} placeholder="support@yourdomain.com" />
    </div>
  </div>

  <!-- Jam operasional -->
  <div class="card">
    <div style="font-weight:500;font-size:14px;margin-bottom:14px">🕐 Jam Operasional</div>
    <input class="input" bind:value={cfg.operational_hours}
      placeholder="Senin - Sabtu, 08.00 - 21.00 WIB" />
  </div>

  <!-- Template pesan komplain -->
  <div class="card">
    <div style="font-weight:500;font-size:14px;margin-bottom:6px">💬 Template Pesan Komplain</div>
    <div style="font-size:12.5px;color:var(--text-muted);margin-bottom:10px">
      Template ini diisi otomatis di form komplain WhatsApp. Variabel: <code>{'{invoice_no}'}</code> <code>{'{product_name}'}</code> <code>{'{buyer_name}'}</code> <code>{'{buyer_email}'}</code> <code>{'{phone}'}</code> <code>{'{status}'}</code> <code>{'{issue}'}</code>
    </div>
    <textarea class="input" rows="5" style="resize:vertical" bind:value={cfg.complaint_template}
      placeholder={`Halo admin, saya ingin komplain order berikut:\n\nInvoice: {invoice_no}\nProduk: {product_name}\nNama: {buyer_name}\nEmail: {buyer_email}\nNomor HP: {phone}\nStatus: {status}\n\nMasalah:\n{issue}`}></textarea>
    <div style="font-size:12px;color:var(--text-muted);margin-top:6px">
      Template ini dikirim via WhatsApp saat pembeli klik tombol komplain.
    </div>
  </div>

  {#if error}<div class="alert-error">{error}</div>{/if}

  <div style="display:flex;align-items:center;gap:12px">
    <button class="btn btn-primary" on:click={save} disabled={saving}>
      {saving?'Menyimpan…':'Simpan Konfigurasi'}
    </button>
    {#if saved}<span style="color:#2f5e0f;font-size:13px;font-weight:500">✓ Tersimpan!</span>{/if}
  </div>
</div>
{/if}

<script>
  import { onMount } from 'svelte';
  import { api } from '$lib/api.js';

  let cfg = null;
  let loading = true;
  let saving = false;
  let saved = false;
  let error = '';

  onMount(async () => {
    try { cfg = await api.getAdminPayment(); }
    catch(e) { error = e.message; }
    finally { loading = false; }
  });

  async function save() {
    saving = true; error = ''; saved = false;
    try {
      cfg = await api.updatePaymentConfig(cfg);
      saved = true; setTimeout(() => saved = false, 2500);
    } catch(e) { error = e.message; }
    finally { saving = false; }
  }

  // Hanya satu gateway yang boleh aktif bersamaan
  function onSayaBayarToggle() {
    if (cfg.sayabayar_enabled) cfg.dompetx_enabled = false;
  }
  function onDompetXToggle() {
    if (cfg.dompetx_enabled) cfg.sayabayar_enabled = false;
  }

  $: activeGateway = cfg?.sayabayar_enabled ? 'sayabayar'
                   : cfg?.dompetx_enabled   ? 'dompetx'
                   : 'manual';
</script>

<svelte:head><title>Pembayaran — DigiStore Admin</title></svelte:head>

<div class="page-header"><h1 class="page-title">Pengaturan Pembayaran</h1></div>

{#if loading}
  <div style="color:var(--text-muted);padding:2rem">Memuat…</div>
{:else if cfg}
<div style="display:flex;flex-direction:column;gap:14px">

  <!-- Status aktif -->
  <div class="status-bar">
    <span style="font-size:13px;color:var(--text-muted)">Gateway aktif:</span>
    {#if activeGateway === 'sayabayar'}
      <span class="badge badge-active">⚡ SayaBayar</span>
    {:else if activeGateway === 'dompetx'}
      <span class="badge badge-stock">⚡ DompetX</span>
    {:else}
      <span class="badge badge-inactive">Manual (tanpa gateway)</span>
    {/if}
    <span style="font-size:12px;color:var(--text-muted);margin-left:4px">
      — hanya satu gateway yang bisa aktif bersamaan
    </span>
  </div>

  <!-- ── SayaBayar ──────────────────────────────────────────────────── -->
  <div class="gw-card {cfg.sayabayar_enabled?'gw-active':''}">
    <div class="gw-header">
      <div>
        <div style="font-weight:500;font-size:14.5px">⚡ SayaBayar</div>
        <div style="font-size:12px;color:var(--text-muted);margin-top:2px">
          Payment gateway Indonesia — satu API untuk semua metode bayar
        </div>
      </div>
      <label class="switch">
        <input type="checkbox" bind:checked={cfg.sayabayar_enabled} on:change={onSayaBayarToggle}/>
        <span class="slider"></span>
      </label>
    </div>

    {#if cfg.sayabayar_enabled}
      <div style="display:flex;flex-direction:column;gap:12px;margin-top:14px;padding-top:14px;border-top:0.5px solid var(--border)">
        <div class="form-row-2">
          <div>
            <label class="field-label">API Key *</label>
            <input class="input mono" bind:value={cfg.sayabayar_api_key}
              placeholder="sk_live_xxxxxxxxxxxx" type="password" autocomplete="off"/>
            <div style="font-size:11px;color:var(--text-muted);margin-top:3px">
              Generate di dashboard SayaBayar → API Keys
            </div>
          </div>
          <div>
            <label class="field-label">Channel Preference</label>
            <select class="input" bind:value={cfg.sayabayar_channel}>
              <option value="platform">platform — SayaBayar pilih metode terbaik</option>
              <option value="client">client — dana langsung ke rekening (plan berbayar)</option>
            </select>
          </div>
        </div>

        <div style="display:flex;gap:16px;flex-wrap:wrap">
          <label style="display:flex;align-items:center;gap:8px;cursor:pointer;font-size:13.5px">
            <input type="checkbox" bind:checked={cfg.sayabayar_auto_qris}/>
            Otomatis pilih QRIS
          </label>
          <label style="display:flex;align-items:center;gap:8px;cursor:pointer;font-size:13.5px">
            <input type="checkbox" bind:checked={cfg.sayabayar_auto_confirm}/>
            Otomatis confirm barcode
          </label>
        </div>

        <div style="max-width:280px">
          <label class="field-label">Batas Waktu Pembayaran (jam)</label>
          <input class="input" type="number" min="1" max="168" bind:value={cfg.payment_expire_hours}/>
          <div style="font-size:11px;color:var(--text-muted);margin-top:3px">
            Gateway QRIS sekarang expired otomatis dalam 30 menit.
          </div>
        </div>

        <div class="webhook-box">
          <div style="font-weight:500;font-size:12.5px;margin-bottom:5px">🔗 URL Webhook SayaBayar</div>
          <div style="font-size:12px;color:var(--text-muted);margin-bottom:6px">
            Daftarkan URL ini di dashboard SayaBayar → Webhooks agar pembayaran terkonfirmasi otomatis:
          </div>
          <code class="webhook-url">https://yourdomain.com/api/webhook/sayabayar</code>
          <div style="font-size:11.5px;color:var(--text-muted);margin-top:6px">
            Ganti <code>yourdomain.com</code> dengan domain production kamu.
            Saat development gunakan ngrok: <code>ngrok http 8080</code>
          </div>
        </div>
      </div>
    {/if}
  </div>

  <!-- ── DompetX ────────────────────────────────────────────────────── -->
  <div class="gw-card {cfg.dompetx_enabled?'gw-active':''}">
    <div class="gw-header">
      <div>
        <div style="font-weight:500;font-size:14.5px">🔷 DompetX</div>
        <div style="font-size:12px;color:var(--text-muted);margin-top:2px">
          Payment gateway & aggregator Indonesia — VA, QRIS, e-Wallet, kartu
        </div>
      </div>
      <label class="switch">
        <input type="checkbox" bind:checked={cfg.dompetx_enabled} on:change={onDompetXToggle}/>
        <span class="slider"></span>
      </label>
    </div>

    {#if cfg.dompetx_enabled}
      <div style="display:flex;flex-direction:column;gap:12px;margin-top:14px;padding-top:14px;border-top:0.5px solid var(--border)">
        <div class="form-row-2">
          <div>
            <label class="field-label">API Key *</label>
            <input class="input mono" bind:value={cfg.dompetx_api_key}
              placeholder="dompetx_live_xxxx" type="password" autocomplete="off"/>
          </div>
          <div>
            <label class="field-label">Secret Key (verifikasi webhook) *</label>
            <input class="input mono" bind:value={cfg.dompetx_secret_key}
              placeholder="••••••••••••" type="password" autocomplete="off"/>
          </div>
        </div>
        <div class="form-row-2">
          <div>
            <label class="field-label">Batas Waktu Pembayaran (jam)</label>
            <input class="input" type="number" min="1" max="72" bind:value={cfg.payment_expire_hours}/>
          </div>
          <div>
            <label class="field-label">Mode</label>
            <select class="input" bind:value={cfg.dompetx_sandbox}>
              <option value={true}>Sandbox (Testing)</option>
              <option value={false}>Production (Live)</option>
            </select>
          </div>
        </div>
        <div class="webhook-box">
          <div style="font-weight:500;font-size:12.5px;margin-bottom:5px">🔗 URL Webhook DompetX</div>
          <code class="webhook-url">https://yourdomain.com/api/webhook/dompetx</code>
        </div>
      </div>
    {/if}
  </div>

  <!-- ── Manual Payment ────────────────────────────────────────────── -->
  <div class="gw-card {activeGateway==='manual'?'gw-active':''}">
    <div class="gw-header">
      <div>
        <div style="font-weight:500;font-size:14.5px">🏦 Pembayaran Manual</div>
        <div style="font-size:12px;color:var(--text-muted);margin-top:2px">
          Aktif otomatis jika tidak ada gateway yang dipilih — admin konfirmasi manual di halaman Pesanan
        </div>
      </div>
      <span class="badge {activeGateway==='manual'?'badge-active':'badge-inactive'}" style="font-size:11.5px">
        {activeGateway==='manual'?'Aktif':'Nonaktif'}
      </span>
    </div>

    <div style="margin-top:14px;padding-top:14px;border-top:0.5px solid var(--border);display:flex;flex-direction:column;gap:12px">
      <div class="form-row-3">
        <div><label class="field-label">Nama Bank</label><input class="input" bind:value={cfg.bank_name} placeholder="BCA, Mandiri…"/></div>
        <div><label class="field-label">Nomor Rekening</label><input class="input" bind:value={cfg.bank_no} placeholder="0000000000"/></div>
        <div><label class="field-label">Atas Nama</label><input class="input" bind:value={cfg.bank_acc}/></div>
      </div>
      <div class="form-row-3">
        <div><label class="field-label">DANA</label><input class="input" bind:value={cfg.dana} placeholder="08xxxxxxxxxx"/></div>
        <div><label class="field-label">GoPay</label><input class="input" bind:value={cfg.gopay} placeholder="08xxxxxxxxxx"/></div>
        <div><label class="field-label">OVO</label><input class="input" bind:value={cfg.ovo} placeholder="08xxxxxxxxxx"/></div>
      </div>
      <div style="display:flex;gap:16px">
        <label style="display:flex;align-items:center;gap:8px;cursor:pointer;font-size:13.5px">
          <input type="checkbox" bind:checked={cfg.qris}/>QRIS
        </label>
        <label style="display:flex;align-items:center;gap:8px;cursor:pointer;font-size:13.5px">
          <input type="checkbox" bind:checked={cfg.crypto}/>Cryptocurrency
        </label>
      </div>
      {#if cfg.crypto}
        <div><label class="field-label">Alamat Wallet</label>
          <input class="input mono" bind:value={cfg.crypto_addr} placeholder="bc1q…"/></div>
      {/if}
      <div style="font-size:11px;color:var(--text-muted)">
        Gateway DompetX dan SayaBayar mengikuti expired 30 menit untuk QRIS.
      </div>
    </div>
  </div>

  <!-- SMTP reminder -->
  <div style="background:#f8f8f6;border-radius:var(--radius);padding:11px 14px;font-size:13px;color:var(--text-muted)">
    📧 Email invoice dikonfigurasi via <code>.env</code> backend:
    <code>SMTP_HOST</code> · <code>SMTP_USER</code> · <code>SMTP_PASS</code>
  </div>

  {#if error}<div class="alert-error">{error}</div>{/if}

  <div style="display:flex;align-items:center;gap:12px">
    <button class="btn btn-primary" on:click={save} disabled={saving}>
      {saving?'Menyimpan…':'Simpan Pengaturan'}
    </button>
    {#if saved}<span style="color:#2f5e0f;font-size:13px;font-weight:500">✓ Tersimpan!</span>{/if}
  </div>
</div>
{/if}

<style>
.status-bar {
  display:flex;align-items:center;gap:8px;
  padding:10px 14px;background:var(--color-background-secondary,#f8f8f6);
  border-radius:var(--radius);font-size:13px;
}
.gw-card {
  border:0.5px solid var(--border);border-radius:var(--radius-lg);
  padding:1.15rem 1.25rem;background:#fff;
  transition:border-color 0.15s;
}
.gw-active { border-color:#0d5fa8;border-width:1.5px; }
.gw-header { display:flex;justify-content:space-between;align-items:flex-start;gap:12px; }

/* Toggle switch */
.switch { position:relative;display:inline-block;width:42px;height:24px;flex-shrink:0; }
.switch input { opacity:0;width:0;height:0; }
.slider {
  position:absolute;cursor:pointer;inset:0;
  background:#ccc;border-radius:24px;transition:.2s;
}
.slider:before {
  content:'';position:absolute;width:18px;height:18px;
  left:3px;bottom:3px;background:#fff;
  border-radius:50%;transition:.2s;
}
input:checked + .slider { background:#0d5fa8; }
input:checked + .slider:before { transform:translateX(18px); }

.webhook-box {
  background:#f0f6fd;border-radius:var(--radius);
  padding:11px 14px;font-size:12.5px;color:#185FA5;
}
.webhook-url {
  display:block;
  font-family:'JetBrains Mono',monospace;font-size:12px;
  background:#dceeff;color:#0d5fa8;
  padding:5px 10px;border-radius:5px;margin-top:4px;
  word-break:break-all;
}

@media (max-width: 900px) {
  .status-bar { flex-wrap:wrap; gap:6px; }
  .gw-header { flex-direction:column; align-items:flex-start; }
  .gw-card { padding:1rem; }
  .webhook-box { padding:10px 12px; }
  .form-row-2, .form-row-3 { grid-template-columns:1fr; }
}

@media (max-width: 640px) {
  .status-bar { font-size:12px; }
  .gw-card { border-radius:16px; }
  .gw-card .form-row-2, .gw-card .form-row-3 { gap:10px; }
  .webhook-url { font-size:11px; }
}
</style>

<script>
  import { onMount } from 'svelte';
  import { api } from '$lib/api.js';
  import { IDR, fmtDateTime } from '$lib/utils.js';

  let providers = [];
  let products  = [];
  let pullLogs  = [];
  let loading   = true;
  let form      = null;
  let saving    = false;
  let formError = '';
  let pulling   = null; // provider ID yang sedang di-pull
  let pullResult = null;
  let activeTab = 'providers'; // 'providers' | 'logs'

  onMount(load);

  async function load() {
    loading = true;
    try {
      [providers, products, pullLogs] = await Promise.all([
        api.getProviders(),
        api.adminProducts(),
        api.getAllPullLogs(),
      ]);
    } finally { loading = false; }
  }

  // ── Form ───────────────────────────────────────────────────────────────────
  function openNew() {
    form = {
      name: '', product_id: products[0]?.id || 0, type: 'http_api',
      api_url: '', api_method: 'GET', api_headers: '{}', api_body: '',
      items_path: 'data', item_field: 'key', active: true
    };
    formError = '';
  }
  function openEdit(p) { form = { ...p }; formError = ''; }
  function closeForm() { form = null; formError = ''; }

  async function save() {
    formError = '';
    if (!form.name.trim()) { formError = 'Nama provider wajib diisi.'; return; }
    if (!form.api_url.trim()) { formError = 'URL wajib diisi.'; return; }
    if (!form.product_id) { formError = 'Pilih produk tujuan.'; return; }
    saving = true;
    try {
      const payload = { ...form, product_id: Number(form.product_id) };
      if (form.id) {
        const u = await api.updateProvider(form.id, payload);
        providers = providers.map(p => p.id === form.id ? u : p);
      } else {
        const c = await api.createProvider(payload);
        providers = [c, ...providers];
      }
      closeForm();
    } catch(e) { formError = e.message; }
    finally { saving = false; }
  }

  async function del(p) {
    if (!confirm(`Hapus provider "${p.name}"?`)) return;
    await api.deleteProvider(p.id);
    providers = providers.filter(x => x.id !== p.id);
  }

  // ── Pull ──────────────────────────────────────────────────────────────────
  async function pull(p) {
    pulling = p.id; pullResult = null;
    try {
      const res = await api.pullFromProvider(p.id);
      pullResult = { provider: p.name, ...res };
      // Reload logs
      pullLogs = await api.getAllPullLogs();
      // Update provider last_pull info
      providers = providers.map(x => x.id === p.id
        ? { ...x, last_pull_at: new Date().toISOString(), last_count: res.count } : x);
    } catch(e) {
      pullResult = { provider: p.name, status: 'failed', message: e.message, count: 0 };
    } finally { pulling = null; }
  }

  // ── Test connection (preview 3 items) ─────────────────────────────────────
  let testResult = null;
  let testing = false;
  async function testConnection() {
    testing = true; testResult = null;
    try {
      const res = await api.pullFromProvider(form.id);
      testResult = res;
    } catch(e) {
      testResult = { status: 'failed', message: e.message };
    } finally { testing = false; }
  }

  $: productName = (id) => products.find(p => p.id === Number(id))?.name || '-';
</script>

<svelte:head><title>Provider Stok — Digitalkuh Murah Admin</title></svelte:head>

<div class="page-header">
  <h1 class="page-title">🔌 Provider Stok</h1>
  <button class="btn btn-primary" on:click={openNew}>+ Tambah Provider</button>
</div>

<p style="font-size:13px;color:var(--text-muted);margin-bottom:1.25rem">
  Hubungkan produk ke API eksternal untuk mengisi stok otomatis.
  Sistem menarik item dari provider dan menyimpannya ke tabel stok produk (duplikat dilewati otomatis).
</p>

<!-- Tabs -->
<div style="display:flex;gap:6px;margin-bottom:16px;border-bottom:0.5px solid var(--border);padding-bottom:0">
  {#each [['providers','Providers'],['logs','Pull Logs']] as [t,l]}
    <button class="tab-btn {activeTab===t?'active':''}" on:click={()=>activeTab=t}>{l}</button>
  {/each}
</div>

{#if pullResult}
  <div style="margin-bottom:14px;padding:12px 14px;border-radius:var(--radius);background:{pullResult.status==='success'?'#EAF3DE':pullResult.status==='partial'?'#FAEEDA':'#FCEBEB'};color:{pullResult.status==='success'?'#2f5e0f':pullResult.status==='partial'?'#854F0B':'#8c2626'};font-size:13px">
    <strong>{pullResult.provider}:</strong> {pullResult.message}
    {#if pullResult.count > 0}<span style="margin-left:6px">· {pullResult.count} item ditambahkan</span>{/if}
    <button style="float:right;background:none;border:none;cursor:pointer;color:inherit;font-size:16px" on:click={()=>pullResult=null}>×</button>
  </div>
{/if}

<!-- ── PROVIDERS TAB ─────────────────────────────────────────────────────── -->
{#if activeTab === 'providers'}
  {#if loading}
    <div style="color:var(--text-muted);padding:2rem">Memuat…</div>
  {:else if providers.length === 0}
    <div class="card" style="text-align:center;padding:3rem;color:var(--text-muted)">
      <div style="font-size:40px;margin-bottom:12px">🔌</div>
      <div style="font-weight:500;margin-bottom:8px">Belum ada provider</div>
      <p style="font-size:13px;margin-bottom:16px">Tambah provider untuk mengambil stok dari API eksternal secara otomatis.</p>
      <button class="btn btn-primary" on:click={openNew}>+ Tambah Provider Pertama</button>
    </div>
  {:else}
    <div style="display:flex;flex-direction:column;gap:12px">
      {#each providers as p (p.id)}
        <div class="card" style="display:flex;align-items:flex-start;gap:14px">
          <!-- Icon type -->
          <div style="width:40px;height:40px;border-radius:var(--radius);background:{p.type==='http_api'?'#E6F1FB':'#EAF3DE'};display:flex;align-items:center;justify-content:center;font-size:18px;flex-shrink:0">
            {p.type === 'http_api' ? '🔗' : '📄'}
          </div>

          <div style="flex:1;min-width:0">
            <div style="display:flex;align-items:center;gap:8px;margin-bottom:4px;flex-wrap:wrap">
              <span style="font-weight:500;font-size:14.5px">{p.name}</span>
              <span class="badge {p.active?'badge-active':'badge-inactive'}">{p.active?'Aktif':'Nonaktif'}</span>
              <span class="badge badge-stock">{p.type === 'http_api' ? 'HTTP API' : 'CSV URL'}</span>
            </div>
            <div style="font-size:12.5px;color:var(--text-muted);margin-bottom:4px">
              → Produk: <strong>{productName(p.product_id)}</strong>
            </div>
            <div class="mono" style="font-size:11.5px;color:var(--text-muted);word-break:break-all;margin-bottom:4px">
              {p.api_method} {p.api_url}
            </div>
            {#if p.items_path || p.item_field}
              <div style="font-size:11.5px;color:var(--text-muted)">
                Path: <code>{p.items_path || 'root'}</code>
                {#if p.item_field} · Field: <code>{p.item_field}</code>{/if}
              </div>
            {/if}
            {#if p.last_pull_at}
              <div style="font-size:11.5px;color:var(--text-muted);margin-top:4px">
                Pull terakhir: {fmtDateTime(p.last_pull_at)} · {p.last_count} item
              </div>
            {:else}
              <div style="font-size:11.5px;color:var(--text-muted);margin-top:4px">Belum pernah di-pull</div>
            {/if}
          </div>

          <div style="display:flex;flex-direction:column;gap:6px;flex-shrink:0">
            <button class="btn btn-sm" style="background:#EAF3DE;color:#2f5e0f;border-color:#c0dda8;white-space:nowrap"
              disabled={pulling === p.id} on:click={()=>pull(p)}>
              {pulling === p.id ? '⏳ Menarik…' : '⬇ Pull Sekarang'}
            </button>
            <button class="btn btn-sm" on:click={()=>openEdit(p)}>Edit</button>
            <button class="btn btn-sm btn-danger" on:click={()=>del(p)}>Hapus</button>
          </div>
        </div>
      {/each}
    </div>
  {/if}

<!-- ── LOGS TAB ─────────────────────────────────────────────────────────── -->
{:else}
  <div class="card" style="padding:0;overflow:hidden">
    <div style="overflow-x:auto">
      <table class="data-table">
        <thead>
          <tr><th>Waktu</th><th>Produk</th><th>Status</th><th>Item</th><th>Pesan</th></tr>
        </thead>
        <tbody>
          {#each pullLogs as log (log.id)}
            <tr>
              <td style="white-space:nowrap;font-size:12px;color:var(--text-muted)">{fmtDateTime(log.created_at)}</td>
              <td style="font-size:13px">{productName(log.product_id)}</td>
              <td><span class="badge badge-{log.status==='success'?'active':log.status==='partial'?'pending':'failed'}">
                {log.status}
              </span></td>
              <td style="font-weight:500;text-align:center">{log.count}</td>
              <td style="font-size:12.5px;color:var(--text-muted);max-width:300px">{log.message}</td>
            </tr>
          {/each}
          {#if pullLogs.length === 0}
            <tr><td colspan="5" style="text-align:center;padding:2rem;color:var(--text-muted)">Belum ada pull log.</td></tr>
          {/if}
        </tbody>
      </table>
    </div>
  </div>
{/if}

<!-- ════════ PROVIDER FORM MODAL ════════ -->
{#if form !== null}
  <div class="modal-overlay" on:click={(e)=>e.target===e.currentTarget&&closeForm()} role="dialog">
    <div class="modal-box" style="max-width:600px">
      <div class="modal-header">
        <span class="modal-title">{form.id?'Edit Provider':'Tambah Provider Baru'}</span>
        <button class="modal-close" on:click={closeForm}>×</button>
      </div>

      <div style="display:flex;flex-direction:column;gap:14px">
        <div class="form-row-2">
          <div>
            <label class="field-label">Nama Provider *</label>
            <input class="input" bind:value={form.name} placeholder="mis: Stok Supplier A" />
          </div>
          <div>
            <label class="field-label">Produk Tujuan *</label>
            <select class="input" bind:value={form.product_id}>
              {#each products.filter(p=>p.type==='stock') as p}
                <option value={p.id}>{p.name}</option>
              {/each}
            </select>
          </div>
        </div>

        <div>
          <label class="field-label">Tipe Provider</label>
          <div style="display:flex;gap:8px">
            {#each [['http_api','🔗 HTTP API','Ambil dari REST API JSON'],['csv_url','📄 CSV URL','Ambil dari file CSV/text, satu item per baris']] as [t,l,d]}
              <div class="type-card {form.type===t?'selected':''}" on:click={()=>form.type=t} role="button" tabindex="0" on:keydown={(e)=>e.key==='Enter'&&(form.type=t)}>
                <div style="font-weight:500;font-size:13px">{l}</div>
                <div style="font-size:12px;color:var(--text-muted);margin-top:3px">{d}</div>
              </div>
            {/each}
          </div>
        </div>

        <div>
          <label class="field-label">URL *</label>
          <input class="input mono" bind:value={form.api_url} placeholder="https://api.supplier.com/v1/stocks" />
        </div>

        {#if form.type === 'http_api'}
          <div class="form-row-2">
            <div>
              <label class="field-label">Method</label>
              <select class="input" bind:value={form.api_method}>
                <option>GET</option><option>POST</option><option>PUT</option>
              </select>
            </div>
            <div>
              <label class="field-label">Headers (JSON)</label>
              <input class="input mono" bind:value={form.api_headers}
                placeholder='{{"X-API-Key":"sk_xxx","Authorization":"Bearer token"}}' />
            </div>
          </div>

          {#if form.api_method !== 'GET'}
            <div>
              <label class="field-label">Request Body (JSON)</label>
              <textarea class="input mono" rows="3" style="resize:vertical" bind:value={form.api_body}
                placeholder='{{"limit":100,"page":1}}'></textarea>
            </div>
          {/if}

          <div style="background:#f8f8f6;border-radius:var(--radius);padding:14px">
            <div style="font-weight:500;font-size:13px;margin-bottom:10px">📍 Konfigurasi Parsing Response</div>
            <div class="form-row-2">
              <div>
                <label class="field-label">Items Path (dot notation)</label>
                <input class="input mono" bind:value={form.items_path}
                  placeholder="data.items" />
                <div style="font-size:11px;color:var(--text-muted);margin-top:3px">
                  Path ke array item. Kosongkan jika response langsung array.
                </div>
              </div>
              <div>
                <label class="field-label">Item Field</label>
                <input class="input mono" bind:value={form.item_field}
                  placeholder="key" />
                <div style="font-size:11px;color:var(--text-muted);margin-top:3px">
                  Field dari tiap object yang dijadikan nilai stok. Kosongkan jika item adalah string langsung.
                </div>
              </div>
            </div>

            <!-- Contoh JSON mapping -->
            <div style="margin-top:10px;padding:10px 12px;background:#fff;border-radius:var(--radius);font-size:12px;border:0.5px solid var(--border)">
              <div style="color:var(--text-muted);margin-bottom:6px;font-weight:500">Contoh mapping:</div>
              <div style="display:grid;grid-template-columns:1fr 1fr;gap:8px">
                <div>
                  <div style="color:var(--text-muted);font-size:11px;margin-bottom:3px">Response API:</div>
                  <pre style="font-size:11px;color:#534AB7;margin:0">{`{ "data": {
  "items": [
    { "key": "LIC-001" },
    { "key": "LIC-002" }
  ]
}}`}</pre>
                </div>
                <div>
                  <div style="color:var(--text-muted);font-size:11px;margin-bottom:3px">Config:</div>
                  <div style="font-size:11.5px">
                    Items Path: <code style="background:#f0f4ff;padding:1px 5px;border-radius:3px">data.items</code><br/>
                    Item Field: <code style="background:#f0f4ff;padding:1px 5px;border-radius:3px">key</code>
                  </div>
                  <div style="font-size:11px;color:#2f5e0f;margin-top:8px">→ Hasilnya: LIC-001, LIC-002</div>
                </div>
              </div>
            </div>
          </div>
        {:else}
          <!-- CSV: satu item per baris -->
          <div style="background:#EAF3DE;border-radius:var(--radius);padding:11px 14px;font-size:13px;color:#2f5e0f">
            📄 Setiap baris di file CSV/text akan menjadi satu item stok. Baris kosong dilewati otomatis.
          </div>
        {/if}

        <div style="display:flex;align-items:center;gap:8px">
          <input type="checkbox" id="pact" bind:checked={form.active} />
          <label for="pact" style="font-size:13.5px;cursor:pointer">Provider aktif</label>
        </div>

        {#if formError}<div class="alert-error">{formError}</div>{/if}

        <div style="display:flex;gap:10px;justify-content:flex-end;border-top:0.5px solid var(--border);padding-top:14px">
          <button class="btn" on:click={closeForm}>Batal</button>
          <button class="btn btn-primary" on:click={save} disabled={saving}>
            {saving ? 'Menyimpan…' : 'Simpan Provider'}
          </button>
        </div>
      </div>
    </div>
  </div>
{/if}

<style>
.tab-btn {
  padding: 8px 16px;
  border: none;
  background: transparent;
  cursor: pointer;
  font-size: 13.5px;
  font-family: inherit;
  color: var(--text-muted);
  border-bottom: 2px solid transparent;
  margin-bottom: -1px;
}
.tab-btn.active { color: #0d5fa8; border-bottom-color: #0d5fa8; font-weight: 500; }
.type-card {
  flex: 1;
  padding: 10px 13px;
  border: 0.5px solid var(--border);
  border-radius: var(--radius);
  cursor: pointer;
  transition: border-color 0.12s;
}
.type-card.selected { border: 1.5px solid #0d5fa8; background: #fafeff; }
</style>

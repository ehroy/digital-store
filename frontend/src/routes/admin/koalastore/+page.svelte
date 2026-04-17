<script>
  import { onMount } from 'svelte';
  import { api } from '$lib/api.js';
  import { IDR, fmtDateTime } from '$lib/utils.js';

  let providers = [];
  let activeProvider = null;
  let balance = null;
  let providerProducts = [];
  let productsMeta = { total: 0, page: 1, total_pages: 1 };
  let loading = true;
  let syncing = false;
  let importingCodes = new Set();
  let syncResult = null;

  // Filter state
  let search = '';
  let filterCat = '';
  let filterImported = '';
  let page = 1;

  // Provider form
  let providerForm = null;
  let savingProvider = false;
  let providerError = '';

  // Import config
  let markupType = 'percent';
  let markupValue = 20;
  let autoSync = true;

  // Selected untuk bulk import
  let selectedCodes = new Set();
  let selectAll = false;

  onMount(load);

  async function load() {
    loading = true;
    try {
      providers = await api.getExtProviders();
      if (providers.length > 0 && !activeProvider) {
        await selectProvider(providers[0]);
      }
    } finally { loading = false; }
  }

  async function selectProvider(p) {
    console.log(p)
    activeProvider = p;
    markupType = p.default_markup_type || 'percent';
    markupValue = p.default_markup_value ?? 20;
    balance = null;
    selectedCodes = new Set();
    selectAll = false;
    await Promise.all([loadBalance(), loadProducts(1)]);
  }

  async function loadBalance() {
    try { balance = await api.getExtProviderBalance(activeProvider.id); } catch {}
  }

  async function loadProducts(p = page) {
    page = p;
    const qs = `?page=${p}&search=${encodeURIComponent(search)}&category=${encodeURIComponent(filterCat)}&imported=${filterImported}`;
    const res = await api.getExtProviderProducts(activeProvider.id, qs);
    providerProducts = res.items || [];
    productsMeta = { total: res.total, page: res.page, total_pages: res.total_pages };
    selectedCodes = new Set();
    selectAll = false;
  }

  async function sync() {
    syncing = true; syncResult = null;
    try {
      syncResult = await api.syncExtProvider(activeProvider.id);
      await loadProducts(1);
    } catch(e) { syncResult = { message: e.message, error: true }; }
    finally { syncing = false; }
  }

  async function importSelected() {
    if (selectedCodes.size === 0) return;
    const codes = [...selectedCodes];
    for (const c of codes) importingCodes = new Set([...importingCodes, c]);
    try {
      const res = await api.importProviderProducts(activeProvider.id, {
        codes, markup_type: markupType, markup_value: Number(markupValue), auto_sync: autoSync
      });
      alert(`✅ ${res.message}`);
      await loadProducts(page);
    } catch(e) { alert('Gagal: ' + e.message); }
    finally {
      for (const c of codes) { importingCodes.delete(c); }
      importingCodes = new Set([...importingCodes]);
      selectedCodes = new Set();
    }
  }

  async function importOne(code) {
    importingCodes = new Set([...importingCodes, code]);
    try {
      await api.importProviderProducts(activeProvider.id, {
        codes: [code], markup_type: markupType, markup_value: Number(markupValue), auto_sync: autoSync
      });
      providerProducts = providerProducts.map(p => p.code === code ? {...p, imported: true} : p);
    } catch(e) { alert('Gagal: ' + e.message); }
    finally { importingCodes.delete(code); importingCodes = new Set([...importingCodes]); }
  }

  async function syncPrices() {
    try {
      const r = await api.syncProviderPrices();
      alert(`✅ ${r.message}`);
    } catch(e) { alert('Gagal: ' + e.message); }
  }

  async function applyDefaultMarkup() {
    if (!activeProvider) return;
    try {
      const r = await api.applyProviderDefaultMarkup(activeProvider.id);
      alert(`✅ ${r.message}`);
      await loadProducts(page);
    } catch(e) { alert('Gagal: ' + e.message); }
  }

  // Toggle select
  function toggleCode(code) {
    const s = new Set(selectedCodes);
    s.has(code) ? s.delete(code) : s.add(code);
    selectedCodes = s;
  }
  function toggleAll() {
    selectAll = !selectAll;
    selectedCodes = selectAll ? new Set(providerProducts.filter(p=>!p.imported).map(p=>p.code)) : new Set();
  }

  // Provider form
  function openNewProvider() {
    providerForm = {
      name:'KoalaStore',
      type:'koalastore',
      base_url:'https://koalastore.digital/api/v1',
      api_key:'',
      default_markup_type:'percent',
      default_markup_value:20,
      active:true,
    };
    providerError = '';
  }
  async function saveProvider() {
    providerError = '';
    if (!providerForm.api_key.trim()) { providerError = 'API Key wajib diisi.'; return; }
    savingProvider = true;
    try {
      if (providerForm.id) {
        await api.updateExtProvider(providerForm.id, providerForm);
      } else {
        const r = await api.createExtProvider(providerForm);
        providers = [r, ...providers];
        await selectProvider(r);
      }
      providerForm = null;
      providers = await api.getExtProviders();
    } catch(e) { providerError = e.message; }
    finally { savingProvider = false; }
  }
  async function delProvider(p) {
    if (!confirm(`Hapus provider "${p.name}"?`)) return;
    await api.deleteExtProvider(p.id);
    providers = providers.filter(x => x.id !== p.id);
    if (activeProvider?.id === p.id) { activeProvider = null; providerProducts = []; balance = null; }
  }

  $: sellPrice = (basePrice) => {
    if (markupType === 'percent') return Math.round(basePrice * (1 + markupValue/100));
    return basePrice + Number(markupValue);
  };

  $: categories = [...new Set(providerProducts.map(p=>p.category).filter(Boolean))];
</script>

<svelte:head><title>KoalaStore — Digital Murah Admin</title></svelte:head>

<div class="page-header">
  <div>
    <h1 class="page-title">🐨 KoalaStore Provider</h1>
    <p style="font-size:13px;color:var(--text-muted);margin-top:2px">
      Sync produk dari KoalaStore → import ke toko Digital Murah dengan markup otomatis.
    </p>
  </div>
  <div style="display:flex;gap:8px">
    <button class="btn btn-sm" on:click={syncPrices} title="Update harga semua produk provider dengan auto_sync=true">
      🔄 Sync Semua Harga
    </button>
    <button class="btn btn-sm" on:click={applyDefaultMarkup} title="Terapkan default markup provider ke semua produk yang sudah diimport">
      📈 Terapkan Markup Default
    </button>
    <button class="btn btn-primary" on:click={openNewProvider}>+ Tambah Provider</button>
  </div>
</div>

{#if loading}
  <div style="color:var(--text-muted);padding:2rem">Memuat…</div>

{:else if providers.length === 0}
  <div class="card" style="text-align:center;padding:3rem">
    <div style="font-size:48px;margin-bottom:14px">🐨</div>
    <div style="font-weight:500;font-size:16px;margin-bottom:8px">Belum ada provider KoalaStore</div>
    <p style="font-size:13px;color:var(--text-muted);margin-bottom:16px">
      Tambahkan provider dengan API key dari dashboard KoalaStore kamu.
    </p>
    <button class="btn btn-primary" on:click={openNewProvider}>+ Tambah Provider KoalaStore</button>
  </div>

{:else}
  <!-- Provider selector & info -->
  <div style="display:flex;gap:12px;margin-bottom:16px;flex-wrap:wrap">
    {#each providers as p}
      <div class="provider-chip {activeProvider?.id===p.id?'active':''}"
        on:click={()=>selectProvider(p)} role="button" tabindex="0" on:keydown={(e)=>e.key==='Enter'&&selectProvider(p)}>
        <span style="font-size:16px">🐨</span>
        <div>
          <div style="font-weight:500;font-size:13px">{p.name}</div>
          <div style="font-size:11px;color:var(--text-muted)">{p.last_sync_at ? fmtDateTime(p.last_sync_at) : 'Belum pernah sync'}</div>
        </div>
        <div style="margin-left:auto;display:flex;gap:4px">
          <button class="btn btn-sm" style="padding:3px 8px;font-size:11px" on:click|stopPropagation={()=>{providerForm={...p};providerError=''}}>Edit</button>
          <button class="btn btn-sm btn-danger" style="padding:3px 8px;font-size:11px" on:click|stopPropagation={()=>delProvider(p)}>×</button>
        </div>
      </div>
    {/each}
  </div>

  {#if activeProvider}
    <!-- Top bar: balance + sync -->
    <div style="display:flex;gap:12px;margin-bottom:16px;flex-wrap:wrap;align-items:stretch">

      <!-- Balance card -->
      <div class="card" style="flex:1;min-width:220px;padding:1rem">
        {#if balance}
          <div style="font-size:11.5px;color:var(--text-muted);margin-bottom:6px">💰 Saldo KoalaStore</div>
          <div style="font-size:24px;font-weight:500;color:#0d5fa8">{balance.formatted_balance || IDR(balance.balance)}</div>
          <div style="font-size:11.5px;color:var(--text-muted);margin-top:4px">Total dibelanjakan: {balance.formatted_total_spent || IDR(balance.total_spent)}</div>
        {:else}
          <div style="color:var(--text-muted);font-size:13px">Memuat saldo…</div>
        {/if}
      </div>

      <!-- Sync button -->
      <div class="card" style="padding:1rem;display:flex;flex-direction:column;justify-content:space-between;min-width:200px">
        <div style="font-size:12px;color:var(--text-muted)">Sinkronisasi produk dari KoalaStore ke cache lokal (tidak membuat produk baru).</div>
        <button class="btn btn-primary" style="margin-top:10px;width:100%" on:click={sync} disabled={syncing}>
          {syncing ? '⏳ Syncing…' : '🔄 Sync Produk Sekarang'}
        </button>
      </div>

      <!-- Import config -->
      <div class="card" style="padding:1rem;min-width:260px">
        <div style="font-size:12.5px;font-weight:500;margin-bottom:10px">⚙️ Konfigurasi Import & Markup</div>
        <div class="form-row-2" style="gap:8px;margin-bottom:8px">
          <div>
            <label class="field-label">Tipe Markup</label>
            <select class="input" bind:value={markupType} style="font-size:12px;padding:6px 10px">
              <option value="percent">Persen (%)</option>
              <option value="fixed">Nominal (Rp)</option>
            </select>
          </div>
          <div>
            <label class="field-label">Nilai</label>
            <input class="input" type="number" min="0" step="0.1" bind:value={markupValue} style="font-size:12px;padding:6px 10px"/>
            <div style="font-size:11px;color:var(--text-muted);margin-top:3px">Contoh: 15, 20, atau 20.5</div>
          </div>
        </div>
        <label style="display:flex;align-items:center;gap:7px;cursor:pointer;font-size:12.5px">
          <input type="checkbox" bind:checked={autoSync}/>
          Harga update otomatis saat sync
        </label>
      </div>
    </div>

    {#if syncResult}
      <div style="margin-bottom:12px;padding:10px 14px;border-radius:var(--radius);background:{syncResult.error?'#FCEBEB':'#EAF3DE'};color:{syncResult.error?'#8c2626':'#2f5e0f'};font-size:13px">
        {syncResult.message}
        {#if !syncResult.error}<span style="margin-left:8px">· {syncResult.added} baru · {syncResult.updated} diperbarui</span>{/if}
        <button style="float:right;background:none;border:none;cursor:pointer;color:inherit" on:click={()=>syncResult=null}>×</button>
      </div>
    {/if}

    <!-- Bulk import toolbar -->
    {#if selectedCodes.size > 0}
      <div class="bulk-bar">
        <span style="font-size:13px;font-weight:500">{selectedCodes.size} produk dipilih</span>
        <div style="display:flex;align-items:center;gap:8px">
          <span style="font-size:12px;color:var(--text-muted)">
            Markup {markupType==='percent'?markupValue+'%':'Rp '+markupValue}
          </span>
          <button class="btn btn-success" style="padding:6px 16px" on:click={importSelected}>
            📥 Import {selectedCodes.size} Produk
          </button>
          <button class="btn btn-sm" on:click={()=>{selectedCodes=new Set();selectAll=false}}>Batal</button>
        </div>
      </div>
    {/if}

    <!-- Filter -->
    <div style="display:flex;gap:8px;margin-bottom:12px;flex-wrap:wrap">
      <input class="input" style="flex:1;min-width:180px" placeholder="🔍 Cari produk…"
        bind:value={search} on:input={()=>loadProducts(1)}/>
      <select class="input" style="max-width:160px" bind:value={filterCat} on:change={()=>loadProducts(1)}>
        <option value="">Semua Kategori</option>
        {#each categories as cat}<option value={cat}>{cat}</option>{/each}
      </select>
      <select class="input" style="max-width:160px" bind:value={filterImported} on:change={()=>loadProducts(1)}>
        <option value="">Semua</option>
        <option value="false">Belum Diimport</option>
        <option value="true">Sudah Diimport</option>
      </select>
    </div>

    <!-- Product table -->
    <div class="card" style="padding:0;overflow:hidden">
      <div style="overflow-x:auto">
        <table class="data-table">
          <thead>
            <tr>
              <th style="width:36px">
                <input type="checkbox" checked={selectAll} on:change={toggleAll}/>
              </th>
              <th>Kode Varian</th>
              <th>Nama Produk</th>
              <th>Kategori</th>
              <th>Harga Beli</th>
              <th>Harga Jual</th>
              <th>Stok</th>
              <th>Status</th>
              <th></th>
            </tr>
          </thead>
          <tbody>
            {#each providerProducts as p (p.id)}
              <tr style="background:{p.imported?'#f9fbf9':'#fff'}">
                <td>
                  {#if !p.imported}
                    <input type="checkbox" checked={selectedCodes.has(p.code)}
                      on:change={()=>toggleCode(p.code)}/>
                  {/if}
                </td>
                <td class="mono" style="font-size:12px">{p.code}</td>
                <td style="max-width:200px">
                  <div style="font-size:13px;font-weight:{p.imported?400:500}">{p.name}</div>
                  {#if p.description}
                    <div style="font-size:11px;color:var(--text-muted)">{p.description.slice(0,60)}</div>
                  {/if}
                </td>
                <td style="font-size:12.5px">{p.category}</td>
                <td style="font-size:13px">{IDR(p.provider_price)}</td>
                <td style="font-size:13px;color:#0d5fa8;font-weight:500">{IDR(sellPrice(p.provider_price))}</td>
                <td>
                  <span class="badge {p.stock==='available'?'badge-active':p.stock==='out_of_stock'?'badge-failed':'badge-pending'}" style="font-size:11px">
                    {p.stock==='available'?'✓ Ada':p.stock==='out_of_stock'?'✗ Habis':'Manual'}
                  </span>
                </td>
                <td>
                  {#if p.imported}
                    <span class="badge badge-active" style="font-size:11px">✓ Imported</span>
                  {:else}
                    <span class="badge badge-inactive" style="font-size:11px">Belum</span>
                  {/if}
                </td>
                <td>
                  {#if !p.imported}
                    <button class="btn btn-sm" style="font-size:11px;padding:4px 10px;background:#EAF3DE;color:#2f5e0f;border-color:#c0dda8"
                      disabled={importingCodes.has(p.code)} on:click={()=>importOne(p.code)}>
                      {importingCodes.has(p.code)?'⏳':'📥'} Import
                    </button>
                  {:else}
                    <span style="font-size:11px;color:var(--text-muted)">Sudah ada</span>
                  {/if}
                </td>
              </tr>
            {/each}
            {#if providerProducts.length === 0}
              <tr><td colspan="9" style="text-align:center;padding:2.5rem;color:var(--text-muted)">
                {providers.length > 0 ? 'Belum ada produk. Klik "Sync Produk" untuk mengambil dari KoalaStore.' : 'Tidak ada produk.'}
              </td></tr>
            {/if}
          </tbody>
        </table>
      </div>

      <!-- Pagination -->
      {#if productsMeta.total_pages > 1}
        <div style="display:flex;align-items:center;justify-content:space-between;padding:10px 16px;border-top:0.5px solid var(--border);background:#f9f9f9">
          <span style="font-size:12.5px;color:var(--text-muted)">{productsMeta.total} produk total</span>
          <div style="display:flex;gap:6px;align-items:center">
            <button class="btn btn-sm" on:click={()=>loadProducts(page-1)} disabled={page<=1}>← Prev</button>
            <span style="font-size:12.5px">Hal {page} / {productsMeta.total_pages}</span>
            <button class="btn btn-sm" on:click={()=>loadProducts(page+1)} disabled={page>=productsMeta.total_pages}>Next →</button>
          </div>
        </div>
      {/if}
    </div>
  {/if}
{/if}

<!-- Provider form modal -->
{#if providerForm !== null}
  <div class="modal-overlay" on:click={(e)=>e.target===e.currentTarget&&(providerForm=null)} on:keydown={(e)=>e.key==='Escape'&&(providerForm=null)} tabindex="0" role="dialog" aria-modal="true">
    <div class="modal-box" style="max-width:480px">
      <div class="modal-header">
        <span class="modal-title">{providerForm.id?'Edit Provider':'Tambah Provider KoalaStore'}</span>
        <button class="modal-close" on:click={()=>providerForm=null}>×</button>
      </div>
      <div style="display:flex;flex-direction:column;gap:14px">
        <div>
          <label class="field-label">Nama Provider</label>
          <input class="input" bind:value={providerForm.name} placeholder="KoalaStore Production" />
        </div>
        <div>
          <label class="field-label">API Key *</label>
          <input class="input mono" bind:value={providerForm.api_key}
            placeholder="sk_xxxxxxxxxxxxxxxxxxxx" type="password" autocomplete="off"/>
          <div style="font-size:11.5px;color:var(--text-muted);margin-top:3px">
            Dapatkan API key di dashboard KoalaStore → Settings → API
          </div>
        </div>
        <div style="background:#E6F1FB;border-radius:var(--radius);padding:11px 14px;font-size:12.5px;color:#185FA5">
          <strong>Auth:</strong> Header <code>X-API-Key</code><br/>
          <strong>Base URL:</strong> <code>https://koalastore.digital/api/v1</code>
        </div>
        <label style="display:flex;align-items:center;gap:8px;cursor:pointer;font-size:13.5px">
          <input type="checkbox" bind:checked={providerForm.active}/>
          Provider aktif
        </label>
        <div style="font-size:11.5px;color:var(--text-muted)">
          Default markup ini dipakai saat import produk baru.
        </div>
        <div class="form-row-2" style="gap:10px">
          <div>
            <label class="field-label">Default Markup Type</label>
            <select class="input" bind:value={providerForm.default_markup_type}>
              <option value="percent">Persen (%)</option>
              <option value="fixed">Nominal (Rp)</option>
            </select>
          </div>
          <div>
            <label class="field-label">Default Markup Value</label>
            <input class="input" type="number" min="0" step="0.1" bind:value={providerForm.default_markup_value} />
          </div>
        </div>
        {#if providerError}<div class="alert-error">{providerError}</div>{/if}
        <div style="display:flex;gap:10px;justify-content:flex-end;border-top:0.5px solid var(--border);padding-top:14px">
          <button class="btn" on:click={()=>providerForm=null}>Batal</button>
          <button class="btn btn-primary" on:click={saveProvider} disabled={savingProvider}>
            {savingProvider?'Menyimpan…':'Simpan'}
          </button>
        </div>
      </div>
    </div>
  </div>
{/if}

<style>
.provider-chip {
  display: flex; align-items: center; gap: 10px;
  padding: 10px 14px;
  border: 0.5px solid var(--border);
  border-radius: var(--radius-lg);
  cursor: pointer; background: #fff;
  min-width: 200px; flex: 1; max-width: 340px;
  transition: border-color 0.12s;
}
.provider-chip.active { border: 1.5px solid #0d5fa8; background: #fafeff; }
.provider-chip:hover { border-color: #0d5fa8; }

.bulk-bar {
  display: flex; align-items: center; justify-content: space-between;
  padding: 10px 14px; background: #E6F1FB; border-radius: var(--radius);
  margin-bottom: 10px; gap: 10px;
}
</style>

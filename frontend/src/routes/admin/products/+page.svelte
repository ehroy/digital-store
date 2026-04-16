<script>
  import { onMount } from 'svelte';
  import { api } from '$lib/api.js';
  import { IDR, PRODUCT_ICONS } from '$lib/utils.js';

  let products = [];
  let loading = true;
  let form = null;
  let saving = false;
  let formError = '';
  let scriptActions = [];

  // Image upload state
  let imageFile = null;
  let imagePreview = null;  // data URL untuk preview lokal
  let imageUploading = false;
  let imageError = '';
  let dragOver = false;

  // Stock modal
  let stockModal = null;
  let stockItems = [];
  let stockLoading = false;
  let bulkText = '';
  let editingItem = null;
  let stockError = '';
  let stockPage = 1;
  let stockTotalPages = 1;
  let stockFilter = ''; // '' | 'available' | 'sold'
  let stockSummary = { available: 0, sold: 0, total: 0 };

  const PROVIDERS = [
    { id:'email',   label:'Email',   icon:'✉️',  color:'#185FA5', bg:'#E6F1FB' },
    { id:'slack',   label:'Slack',   icon:'💬',  color:'#4A154B', bg:'#f0e6f6' },
    { id:'discord', label:'Discord', icon:'🎮',  color:'#5865F2', bg:'#eef0ff' },
    { id:'webhook', label:'Webhook', icon:'🔗',  color:'#534AB7', bg:'#EEEDFE' },
    { id:'log',     label:'Log',     icon:'📋',  color:'#3B6D11', bg:'#EAF3DE' },
  ];
  const providerStyle = (id) => PROVIDERS.find(p=>p.id===id) || PROVIDERS[3];

  onMount(load);
  async function load() {
    loading = true;
    try { products = await api.adminProducts(); }
    finally { loading = false; }
  }

  // ── Product form ─────────────────────────────────────────────────────────
  function openNew() {
    form = { name:'', description:'', price:0, category:'', type:'stock', icon:'📦', active:true, script:'', image_url:'' };
    scriptActions = [];
    imageFile = null; imagePreview = null; imageError = '';
    formError = '';
  }
  function openEdit(p) {
    form = { ...p };
    try { scriptActions = JSON.parse(p.script || '[]'); } catch { scriptActions = []; }
    imageFile = null;
    imagePreview = p.image_url || null;
    imageError = '';
    formError = '';
  }
  function closeForm() { form = null; formError = ''; scriptActions = []; imageFile = null; imagePreview = null; }

  function syncScript() { form.script = JSON.stringify(scriptActions); }
  function addAction(provider) {
    const base = { provider, enabled: true, label: '' };
    if (provider === 'email')   Object.assign(base, { to:'', subject:'', body:'' });
    if (provider === 'webhook') Object.assign(base, { url:'', method:'POST', headers:{} });
    if (['slack','discord'].includes(provider)) Object.assign(base, { webhook_url:'', message:'' });
    if (provider === 'log')     Object.assign(base, { message:'' });
    scriptActions = [...scriptActions, base];
    syncScript();
  }
  function removeAction(i) { scriptActions = scriptActions.filter((_,j)=>j!==i); syncScript(); }
  function moveAction(i,dir) {
    const arr=[...scriptActions]; const j=i+dir;
    if(j<0||j>=arr.length) return;
    [arr[i],arr[j]]=[arr[j],arr[i]];
    scriptActions=arr; syncScript();
  }

  // ── Image handling ───────────────────────────────────────────────────────
  function onFileChange(e) {
    const f = e.target.files?.[0];
    if (f) pickFile(f);
  }
  function onDrop(e) {
    dragOver = false;
    const f = e.dataTransfer.files?.[0];
    if (f) pickFile(f);
  }
  function pickFile(f) {
    imageError = '';
    if (f.size > 5 * 1024 * 1024) { imageError = 'Ukuran file maksimal 5 MB.'; return; }
    if (!['image/jpeg','image/png','image/webp','image/gif'].includes(f.type)) {
      imageError = 'Format tidak didukung. Gunakan JPEG, PNG, WebP, atau GIF.'; return;
    }
    imageFile = f;
    const reader = new FileReader();
    reader.onload = e => imagePreview = e.target.result;
    reader.readAsDataURL(f);
  }
  function removeImage() {
    imageFile = null;
    imagePreview = null;
    if (form) form.image_url = '';
  }

  // ── Save product ─────────────────────────────────────────────────────────
  async function saveProduct() {
    formError = '';
    if (!form.name.trim())     { formError = 'Nama produk wajib diisi.'; return; }
    if (!form.category.trim()) { formError = 'Kategori wajib diisi.'; return; }
    if (form.price <= 0)       { formError = 'Harga harus lebih dari 0.'; return; }
    syncScript();
    saving = true;
    try {
      const payload = { ...form };
      if (form.type === 'stock') payload.script = '';
      else payload.script = JSON.stringify(scriptActions);

      let saved;
      if (form.id) {
        saved = await api.updateProduct(form.id, payload);
        products = products.map(p => p.id === form.id ? saved : p);
      } else {
        saved = await api.createProduct(payload);
        products = [saved, ...products];
      }

      // Upload gambar jika ada file baru dipilih
      if (imageFile && saved.id) {
        imageUploading = true;
        try {
          const imgRes = await api.uploadProductImage(saved.id, imageFile);
          saved.image_url = imgRes.image_url;
          products = products.map(p => p.id === saved.id ? {...p, image_url: imgRes.image_url} : p);
        } catch(e) {
          formError = 'Produk tersimpan tapi gambar gagal diupload: ' + e.message;
          imageUploading = false;
          saving = false;
          return;
        }
        imageUploading = false;
      }

      closeForm();
    } catch(e) { formError = e.message; }
    finally { saving = false; }
  }

  async function toggle(p) {
    const u = await api.toggleProduct(p.id); products = products.map(x => x.id===p.id ? u : x);
  }
  async function del(p) {
    if (!confirm(`Hapus "${p.name}"?`)) return;
    await api.deleteProduct(p.id); products = products.filter(x => x.id!==p.id);
  }

  // Hapus gambar dari produk yang sudah tersimpan (langsung via API)
  async function delImageExisting(p) {
    if (!confirm('Hapus gambar produk ini?')) return;
    try {
      await api.deleteProductImage(p.id);
      products = products.map(x => x.id===p.id ? {...x, image_url:''} : x);
      if (form?.id === p.id) { form.image_url = ''; imagePreview = null; }
    } catch(e) { alert('Gagal menghapus gambar: ' + e.message); }
  }

  // ── Stock modal ──────────────────────────────────────────────────────────
  async function openStock(p) {
    stockModal = p; stockError = ''; bulkText = ''; editingItem = null;
    stockPage = 1; stockFilter = ''; stockTotalPages = 1;
    await loadStock(1);
  }
  async function loadStock(page = stockPage) {
    stockLoading = true;
    try {
      const r = await api.getStock(stockModal.id, page, stockFilter);
      stockItems = r.items || [];
      stockPage = r.page || 1;
      stockTotalPages = r.total_pages || 1;
      stockSummary = { available: r.available || 0, sold: r.sold || 0, total: r.total || 0 };
    } finally { stockLoading = false; }
  }
  async function setStockFilter(f) { stockFilter = f; stockPage = 1; await loadStock(1); }
  async function stockPrev() { if (stockPage > 1) await loadStock(stockPage - 1); }
  async function stockNext() { if (stockPage < stockTotalPages) await loadStock(stockPage + 1); }
  async function addBulk() {
    const lines = bulkText.split('\n').map(l=>l.trim()).filter(Boolean);
    if (!lines.length) { stockError = 'Tidak ada item.'; return; }
    stockError = '';
    try {
      await api.addStock(stockModal.id, lines);
      bulkText = ''; await loadStock();
      products = products.map(p => p.id===stockModal.id
        ? {...p, available_stock:(p.available_stock||0)+lines.length, total_stock:(p.total_stock||0)+lines.length} : p);
    } catch(e) { stockError = e.message; }
  }
  async function saveEdit() {
    if (!editingItem.data.trim()) { stockError = 'Data tidak boleh kosong.'; return; }
    try { await api.updateStockItem(editingItem.id, editingItem.data); await loadStock(); editingItem = null; }
    catch(e) { stockError = e.message; }
  }
  async function delItem(item) {
    if (!confirm('Hapus item ini?')) return;
    try { await api.deleteStockItem(item.id); await loadStock(); } catch(e) { stockError = e.message; }
  }
  async function resetItem(item) {
    if (!confirm('Reset item ini menjadi available?')) return;
    try { await api.resetStockItem(item.id); await loadStock(); } catch(e) { stockError = e.message; }
  }

  $: available = stockSummary.available;
  $: sold      = stockSummary.sold;

  // File input ref untuk trigger klik dari tombol
  let fileInputRef;
</script>

<svelte:head><title>Produk — Digital Murah Admin</title></svelte:head>

<div class="page-header">
  <h1 class="page-title">Manajemen Produk</h1>
  <button class="btn btn-primary" on:click={openNew}>+ Tambah Produk</button>
</div>

{#if loading}
  <div style="color:var(--text-muted);padding:2rem">Memuat…</div>
{:else}
  <div class="card" style="padding:0;overflow:hidden">
    <div style="overflow-x:auto">
      <table class="data-table">
        <thead>
          <tr><th>Produk</th><th>Kategori</th><th>Tipe</th><th>Harga</th><th>Stok</th><th>Status</th><th>Aksi</th></tr>
        </thead>
        <tbody>
          {#each products as p (p.id)}
            <tr>
              <td>
                <div style="display:flex;align-items:center;gap:10px">
                  <!-- Thumbnail gambar atau icon emoji -->
                  {#if p.image_url}
                    <div class="thumb-wrap">
                      <img src={p.image_url} alt={p.name} class="thumb" />
                      <button class="thumb-del" title="Hapus gambar" on:click|stopPropagation={()=>delImageExisting(p)}>×</button>
                    </div>
                  {:else}
                    <div class="thumb-icon">{p.icon}</div>
                  {/if}
                  <div>
                    <div style="font-weight:500;font-size:13.5px">{p.name}</div>
                    <div style="font-size:11px;color:var(--text-muted);max-width:180px;overflow:hidden;text-overflow:ellipsis;white-space:nowrap">
                      {p.description?.slice(0,55)}{(p.description?.length||0)>55?'…':''}
                    </div>
                  </div>
                </div>
              </td>
              <td>{p.category}</td>
              <td><span class="badge badge-{p.type}">{p.type==='stock'?'Stok':'Script'}</span></td>
              <td style="font-weight:500">{IDR(p.price)}</td>
              <td>
                {#if p.type==='stock'}
                  <div style="font-size:13px">
                    <span style="color:{p.available_stock===0?'#8c2626':p.available_stock<5?'#854F0B':'#2f5e0f'};font-weight:500">
                      {p.available_stock??0} tersedia
                    </span>
                    <span style="color:var(--text-muted)"> / {p.total_stock??0}</span>
                  </div>
                {:else}
                  <span style="font-size:12px;color:var(--text-muted)">∞ (jasa)</span>
                {/if}
              </td>
              <td>
                <button class="badge badge-{p.active?'active':'inactive'}" style="cursor:pointer;border:none" on:click={()=>toggle(p)}>
                  {p.active?'Aktif':'Nonaktif'}
                </button>
              </td>
              <td>
                <div style="display:flex;gap:5px;flex-wrap:wrap">
                  <button class="btn btn-sm" on:click={()=>openEdit(p)}>Edit</button>
                  {#if p.type==='stock'}
                    <button class="btn btn-sm" style="background:#E6F1FB;color:#185FA5;border-color:#B5D4F4" on:click={()=>openStock(p)}>Stok</button>
                  {/if}
                  <button class="btn btn-sm btn-danger" on:click={()=>del(p)}>Hapus</button>
                </div>
              </td>
            </tr>
          {/each}
        </tbody>
      </table>
    </div>
  </div>
{/if}

<!-- ════════ PRODUCT FORM MODAL ════════ -->
{#if form !== null}
  <div class="modal-overlay" on:click={(e)=>e.target===e.currentTarget&&closeForm()} role="dialog">
    <div class="modal-box" style="max-width:640px">
      <div class="modal-header">
        <span class="modal-title">{form.id?'Edit Produk':'Tambah Produk Baru'}</span>
        <button class="modal-close" on:click={closeForm}>×</button>
      </div>

      <!-- ── IMAGE UPLOAD ─────────────────────────────────────────── -->
      <div style="margin-bottom:16px">
        <label class="field-label">Gambar Produk <span style="color:var(--text-muted);font-weight:400">(opsional · JPEG/PNG/WebP/GIF · maks 5 MB)</span></label>

        {#if imagePreview}
          <!-- Preview gambar -->
          <div class="img-preview-wrap">
            <img src={imagePreview} alt="preview" class="img-preview" />
            <div class="img-preview-actions">
              <button class="btn btn-sm" style="background:rgba(0,0,0,0.6);color:#fff;border:none" on:click={()=>fileInputRef.click()}>
                🖼 Ganti
              </button>
              <button class="btn btn-sm btn-danger" style="background:rgba(180,0,0,0.8);border:none" on:click={removeImage}>
                × Hapus
              </button>
            </div>
            {#if imageFile}
              <div class="img-badge">Belum disimpan</div>
            {:else if form.image_url}
              <div class="img-badge" style="background:#EAF3DE;color:#2f5e0f">Tersimpan</div>
            {/if}
          </div>
        {:else}
          <!-- Drop zone -->
          <div
            class="dropzone {dragOver?'drag-active':''}"
            on:dragover|preventDefault={()=>dragOver=true}
            on:dragleave={()=>dragOver=false}
            on:drop|preventDefault={onDrop}
            on:click={()=>fileInputRef.click()}
            role="button" tabindex="0"
            on:keydown={(e)=>e.key==='Enter'&&fileInputRef.click()}
          >
            <div style="font-size:32px;margin-bottom:8px">🖼</div>
            <div style="font-weight:500;font-size:13.5px;margin-bottom:4px">Klik atau drag gambar ke sini</div>
            <div style="font-size:12px;color:var(--text-muted)">JPEG · PNG · WebP · GIF · maks 5 MB</div>
          </div>
        {/if}

        <!-- Hidden file input -->
        <input bind:this={fileInputRef} type="file" accept="image/jpeg,image/png,image/webp,image/gif"
          style="display:none" on:change={onFileChange} />

        {#if imageError}<div class="alert-error" style="margin-top:6px">{imageError}</div>{/if}
      </div>

      <!-- ── Icon (hanya tampil jika tidak ada gambar) ─────────────── -->
      {#if !imagePreview}
        <div style="margin-bottom:14px">
          <label class="field-label">Icon Emoji <span style="color:var(--text-muted);font-weight:400">(ditampilkan jika tidak ada gambar)</span></label>
          <div class="icon-grid">
            {#each PRODUCT_ICONS as ic}
              <button class="icon-btn {form.icon===ic?'selected':''}" on:click={()=>form.icon=ic}>{ic}</button>
            {/each}
          </div>
        </div>
      {/if}

      <!-- ── Fields ──────────────────────────────────────────────────── -->
      <div class="form-row-2" style="margin-bottom:12px">
        <div><label class="field-label">Nama Produk *</label><input class="input" bind:value={form.name} /></div>
        <div><label class="field-label">Kategori *</label><input class="input" bind:value={form.category} placeholder="Template, Ebook, Jasa…" /></div>
      </div>
      <div style="margin-bottom:12px">
        <label class="field-label">Deskripsi</label>
        <textarea class="input" rows="2" style="resize:vertical" bind:value={form.description}></textarea>
      </div>
      <div class="form-row-2" style="margin-bottom:16px">
        <div><label class="field-label">Harga (Rp) *</label><input class="input" type="number" min="0" bind:value={form.price} /></div>
        <div>
          <label class="field-label">Tipe</label>
          <select class="input" bind:value={form.type}>
            <option value="stock">Stok — item individual</option>
            <option value="script">Script — eksekusi provider</option>
          </select>
        </div>
      </div>

      {#if form.type==='stock'}
        <div class="info-box" style="margin-bottom:14px">
          📦 Setelah simpan, klik <strong>Stok</strong> di tabel untuk tambah item (key, link, dll).
        </div>
      {:else}
        <!-- Provider action builder -->
        <div style="margin-bottom:14px">
          <div style="font-size:12.5px;color:var(--text-muted);margin-bottom:8px;font-weight:500">Provider Actions</div>
          <div class="provider-picker">
            {#each PROVIDERS as prov}
              <button class="prov-btn" style="background:{prov.bg};color:{prov.color};border-color:{prov.color}30"
                on:click={()=>addAction(prov.id)}>{prov.icon} + {prov.label}</button>
            {/each}
          </div>

          {#if scriptActions.length === 0}
            <div class="empty-box" style="margin-top:10px">Pilih provider di atas untuk menambahkan action.</div>
          {:else}
            <div style="display:flex;flex-direction:column;gap:8px;margin-top:10px">
              {#each scriptActions as action, i}
                {@const ps = providerStyle(action.provider)}
                <div class="action-card" style="border-left:3px solid {ps.color}">
                  <div class="action-head">
                    <div style="display:flex;align-items:center;gap:8px">
                      <button class="toggle-btn {action.enabled?'on':'off'}"
                        on:click={()=>{action.enabled=!action.enabled;scriptActions=scriptActions;syncScript();}}>
                        {action.enabled?'ON':'OFF'}
                      </button>
                      <span style="background:{ps.bg};color:{ps.color};padding:2px 9px;border-radius:999px;font-size:11.5px;font-weight:500;opacity:{action.enabled?1:0.5}">
                        {ps.icon} {ps.label}
                      </span>
                    </div>
                    <div style="display:flex;gap:4px">
                      <button class="btn btn-sm" style="padding:3px 8px" on:click={()=>moveAction(i,-1)} disabled={i===0}>↑</button>
                      <button class="btn btn-sm" style="padding:3px 8px" on:click={()=>moveAction(i,1)} disabled={i===scriptActions.length-1}>↓</button>
                      <button class="btn btn-sm btn-danger" style="padding:3px 8px" on:click={()=>removeAction(i)}>×</button>
                    </div>
                  </div>
                  <div style="margin-bottom:8px">
                    <label class="field-label">Label (opsional)</label>
                    <input class="input" style="max-width:260px" bind:value={action.label} placeholder="mis: Notif ke designer" on:input={syncScript}/>
                  </div>
                  {#if action.provider==='email'}
                    <div class="form-grid" style="gap:8px">
                      <div><label class="field-label">Kirim ke</label>
                        <input class="input" bind:value={action.to} placeholder="admin@example.com" on:input={syncScript}/></div>
                      <div><label class="field-label">Subject</label>
                        <input class="input" bind:value={action.subject} placeholder={'Order {{invoice_no}} — {{product_name}}'} on:input={syncScript}/></div>
                      <div><label class="field-label">Isi email</label>
                        <textarea class="input" rows="3" bind:value={action.body} placeholder={'Pembeli: {{buyer_name}}\nInvoice: {{invoice_no}}'} on:input={syncScript} style="resize:vertical"></textarea></div>
                    </div>
                  {:else if action.provider==='slack'||action.provider==='discord'}
                    <div class="form-grid" style="gap:8px">
                      <div><label class="field-label">Webhook URL</label>
                        <input class="input" bind:value={action.webhook_url} placeholder="https://hooks.slack.com/services/…" on:input={syncScript}/></div>
                      <div><label class="field-label">Pesan</label>
                        <textarea class="input" rows="2" bind:value={action.message} placeholder={'Order baru: {{invoice_no}} dari {{buyer_name}}'} on:input={syncScript} style="resize:vertical"></textarea></div>
                    </div>
                  {:else if action.provider==='webhook'}
                    <div class="form-row-2">
                      <div><label class="field-label">URL</label><input class="input" bind:value={action.url} placeholder="https://api.example.com/hook" on:input={syncScript}/></div>
                      <div><label class="field-label">Method</label>
                        <select class="input" bind:value={action.method} on:change={syncScript}>
                          <option>POST</option><option>PUT</option><option>PATCH</option>
                        </select></div>
                    </div>
                  {:else if action.provider==='log'}
                    <div><label class="field-label">Pesan log</label>
                      <input class="input" bind:value={action.message} placeholder={'Order {{invoice_no}} diterima dari {{buyer_name}}'} on:input={syncScript}/></div>
                  {/if}
                </div>
              {/each}
            </div>
          {/if}
          <div class="var-ref">
            <span class="var-ref-title">Variabel:</span>
            {#each ['{{invoice_no}}','{{product_name}}','{{buyer_name}}','{{buyer_email}}','{{total}}','{{qty}}'] as v}
              <code class="var-chip">{v}</code>
            {/each}
          </div>
        </div>
      {/if}

      <div style="margin-bottom:14px;display:flex;align-items:center;gap:8px">
        <input type="checkbox" id="act" bind:checked={form.active}/>
        <label for="act" style="font-size:13.5px;cursor:pointer">Produk aktif &amp; tampil di toko</label>
      </div>

      {#if formError}<div class="alert-error" style="margin-bottom:12px">{formError}</div>{/if}

      <div style="display:flex;gap:10px;justify-content:flex-end;border-top:0.5px solid var(--border);padding-top:14px">
        <button class="btn" on:click={closeForm}>Batal</button>
        <button class="btn btn-primary" on:click={saveProduct} disabled={saving||imageUploading}>
          {#if imageUploading}Mengupload gambar…{:else if saving}Menyimpan…{:else}Simpan Produk{/if}
        </button>
      </div>
    </div>
  </div>
{/if}

<!-- ════════ STOCK MODAL ════════ -->
{#if stockModal}
  <div class="modal-overlay" on:click={(e)=>e.target===e.currentTarget&&(stockModal=null)} role="dialog">
    <div class="modal-box" style="max-width:660px">
      <div class="modal-header">
        <div>
          <div class="modal-title">Stok — {stockModal.name}</div>
          <div style="font-size:12px;color:var(--text-muted);margin-top:2px">
            <span style="color:#2f5e0f;font-weight:500">{available} tersedia</span> ·
            <span style="color:#854F0B">{sold} terjual</span> · {stockItems.length} total
          </div>
        </div>
        <button class="modal-close" on:click={()=>stockModal=null}>×</button>
      </div>

      <div class="card" style="background:#f8f8f6;margin-bottom:14px;padding:1rem">
        <label class="field-label">Tambah Item <span style="color:var(--text-muted);font-weight:400">(satu per baris)</span></label>
        <textarea class="input mono" rows="4" style="resize:vertical" bind:value={bulkText}
          placeholder="https://drive.google.com/file/d/AAA/view&#10;LIC-KEY-XXXX-1111"></textarea>
        <button class="btn btn-primary btn-sm" style="margin-top:8px" on:click={addBulk}>
          + Tambah {bulkText.split('\n').filter(l=>l.trim()).length||0} Item
        </button>
      </div>

      {#if stockError}<div class="alert-error" style="margin-bottom:10px">{stockError}</div>{/if}

      <!-- Filter tabs -->
      <div style="display:flex;gap:6px;margin-bottom:10px">
        {#each [['','Semua'],['available','Available'],['sold','Sold']] as [f,l]}
          <button class="btn btn-sm {stockFilter===f?'btn-primary':''}" on:click={()=>setStockFilter(f)}>{l}</button>
        {/each}
        <span style="margin-left:auto;font-size:12px;color:var(--text-muted);align-self:center">
          Hal {stockPage}/{stockTotalPages}
        </span>
      </div>

      {#if stockLoading}
        <div style="color:var(--text-muted);padding:1rem;text-align:center">Memuat…</div>
      {:else if stockItems.length===0}
        <div style="text-align:center;padding:2rem;color:var(--text-muted);font-size:13px">Tidak ada item.</div>
      {:else}
        <div style="border:0.5px solid var(--border);border-radius:var(--radius-lg);overflow:hidden">
          <table class="data-table">
            <thead><tr><th style="width:42px">No</th><th>Data</th><th style="width:100px">Status</th><th style="width:110px"></th></tr></thead>
            <tbody>
              {#each stockItems as item, i (item.id)}
                {@const rowNo = (stockPage - 1) * 25 + i + 1}
                <tr style="background:{item.sold?'#fff8f8':'#fff'}">
                  <td style="color:var(--text-muted);font-size:12px">{rowNo}</td>
                  <td>
                    {#if editingItem?.id===item.id}
                      <input class="input mono" style="font-size:12px;padding:5px 9px"
                        bind:value={editingItem.data} on:keydown={(e)=>e.key==='Enter'&&saveEdit()}/>
                    {:else}
                      <span class="mono" style="font-size:12px;word-break:break-all">{item.data}</span>
                      {#if item.sold}
                        <div style="font-size:10.5px;color:var(--text-muted)">Invoice: <span class="mono">{item.invoice_no}</span></div>
                      {/if}
                    {/if}
                  </td>
                  <td>
                    {#if item.sold}
                      <span class="badge" style="background:#FCEBEB;color:#8c2626;font-size:11px">✗ Sold</span>
                    {:else}
                      <span class="badge" style="background:#EAF3DE;color:#3B6D11;font-size:11px">✓ Available</span>
                    {/if}
                  </td>
                  <td>
                    <div style="display:flex;gap:4px;justify-content:flex-end">
                      {#if item.sold}
                        <button class="btn btn-sm" style="font-size:11px;padding:3px 7px" on:click={()=>resetItem(item)}>Reset</button>
                      {:else if editingItem?.id===item.id}
                        <button class="btn btn-sm btn-success" style="font-size:11px;padding:3px 7px" on:click={saveEdit}>✓</button>
                        <button class="btn btn-sm" style="font-size:11px;padding:3px 7px" on:click={()=>editingItem=null}>✗</button>
                      {:else}
                        <button class="btn btn-sm" style="font-size:11px;padding:3px 7px" on:click={()=>editingItem={id:item.id,data:item.data}}>Edit</button>
                        <button class="btn btn-sm btn-danger" style="font-size:11px;padding:3px 7px" on:click={()=>delItem(item)}>Hapus</button>
                      {/if}
                    </div>
                  </td>
                </tr>
              {/each}
            </tbody>
          </table>
        </div>

        <!-- Pagination controls -->
        {#if stockTotalPages > 1}
          <div style="display:flex;align-items:center;justify-content:center;gap:8px;margin-top:10px">
            <button class="btn btn-sm" on:click={stockPrev} disabled={stockPage<=1}>← Prev</button>
            <span style="font-size:12.5px;color:var(--text-muted)">
              Halaman {stockPage} dari {stockTotalPages}
            </span>
            <button class="btn btn-sm" on:click={stockNext} disabled={stockPage>=stockTotalPages}>Next →</button>
          </div>
        {/if}
      {/if}
    </div>
  </div>
{/if}

<style>
/* Thumbnail di tabel */
.thumb-wrap { position:relative;width:48px;height:48px;flex-shrink:0; }
.thumb { width:48px;height:48px;object-fit:cover;border-radius:var(--radius);border:0.5px solid var(--border); }
.thumb-del {
  position:absolute;top:-5px;right:-5px;
  width:17px;height:17px;border-radius:50%;
  background:#8c2626;color:#fff;font-size:11px;
  display:flex;align-items:center;justify-content:center;
  border:none;cursor:pointer;line-height:1;
  opacity:0;transition:opacity 0.15s;
}
.thumb-wrap:hover .thumb-del { opacity:1; }
.thumb-icon {
  width:48px;height:48px;border-radius:var(--radius);
  background:#f8f8f6;border:0.5px solid var(--border);
  display:flex;align-items:center;justify-content:center;
  font-size:22px;flex-shrink:0;
}

/* Dropzone */
.dropzone {
  border:2px dashed var(--border-md);border-radius:var(--radius-lg);
  padding:2rem;text-align:center;cursor:pointer;
  transition:border-color 0.15s,background 0.15s;
  background:#fafafa;
}
.dropzone:hover,.drag-active { border-color:#0d5fa8;background:#f0f6fd; }

/* Image preview */
.img-preview-wrap {
  position:relative;display:inline-block;
  border-radius:var(--radius-lg);overflow:hidden;
  border:0.5px solid var(--border);
  max-width:100%;
}
.img-preview { display:block;max-height:200px;max-width:100%;object-fit:contain;background:#f8f8f6; }
.img-preview-actions {
  position:absolute;bottom:8px;left:50%;transform:translateX(-50%);
  display:flex;gap:6px;opacity:0;transition:opacity 0.15s;
}
.img-preview-wrap:hover .img-preview-actions { opacity:1; }
.img-badge {
  position:absolute;top:8px;right:8px;
  background:rgba(0,0,0,0.55);color:#fff;
  font-size:11px;padding:3px 8px;border-radius:999px;
}

/* Icon grid */
.icon-grid { display:flex;flex-wrap:wrap;gap:6px; }
.icon-btn { font-size:18px;padding:5px 9px;cursor:pointer;border:0.5px solid var(--border);border-radius:var(--radius);background:#f8f8f6; }
.icon-btn.selected { border:2px solid #0d5fa8; }

/* Provider picker */
.provider-picker { display:flex;flex-wrap:wrap;gap:7px; }
.prov-btn { padding:6px 14px;border-radius:999px;border:1px solid;cursor:pointer;font-size:13px;font-family:inherit;font-weight:500; }

/* Action card */
.action-card { background:#fafafa;border:0.5px solid var(--border);border-radius:var(--radius-lg);padding:12px 14px; }
.action-head { display:flex;justify-content:space-between;align-items:center;margin-bottom:10px; }
.toggle-btn { padding:3px 10px;border-radius:999px;font-size:11px;font-weight:700;cursor:pointer;border:1.5px solid;font-family:inherit;letter-spacing:0.5px; }
.toggle-btn.on  { background:#EAF3DE;color:#2f5e0f;border-color:#2f5e0f40; }
.toggle-btn.off { background:#f4f4f2;color:#999;border-color:#ccc; }

/* Misc */
.var-ref { display:flex;flex-wrap:wrap;align-items:center;gap:6px;margin-top:10px;padding:8px 12px;background:#f8f8f6;border-radius:var(--radius); }
.var-ref-title { font-size:11.5px;color:var(--text-muted); }
.var-chip { font-family:'JetBrains Mono',monospace;font-size:11px;background:#E6F1FB;color:#185FA5;padding:2px 7px;border-radius:4px; }
.info-box { background:#E6F1FB;border-radius:var(--radius);padding:10px 14px;font-size:13px;color:#185FA5; }
.empty-box { background:#f8f8f6;border:1px dashed var(--border-md);border-radius:var(--radius);padding:14px;text-align:center;font-size:13px;color:var(--text-muted); }

@media (max-width: 900px) {
  .provider-picker { flex-direction:column; }
  .prov-btn { width:100%; }
  .action-head { flex-direction:column; align-items:flex-start; }
  .action-head > div:last-child { width:100%; display:flex; justify-content:flex-start; flex-wrap:wrap; }
  .icon-grid { gap:8px; }
  .icon-btn { flex:1 0 calc(20% - 6px); min-width:42px; }
  .thumb-wrap, .thumb-icon { width:42px; height:42px; }
  .thumb { width:42px; height:42px; }
  .dropzone { padding:1.5rem 1rem; }
  .img-preview-actions { opacity:1; bottom:6px; }
}

@media (max-width: 640px) {
  .provider-picker { gap:8px; }
  .prov-btn { font-size:12px; padding:7px 10px; }
  .action-card { padding:10px 11px; }
  .action-head { gap:10px; }
  .action-head > div:last-child { gap:6px; }
  .var-ref { padding:8px 10px; }
  .var-chip { font-size:10.5px; }
  .icon-btn { flex:1 0 calc(25% - 6px); }
}
</style>

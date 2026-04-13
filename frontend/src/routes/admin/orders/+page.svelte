<script>
  import { onMount } from 'svelte';
  import { api } from '$lib/api.js';
  import { IDR, fmtDateTime, STATUS_LABEL, PAY_LABEL } from '$lib/utils.js';
  import InvoiceModal from '../../../routes/InvoiceModal.svelte';

  let orders = [];
  let loading = true;
  let filterStatus = 'all';
  let viewOrder = null;
  let deliverModal = null;  // order yang sedang di-deliver manual
  let deliverItems = '';    // textarea: item manual
  let deliverNote = '';
  let delivering = false;
  let deliverError = '';
  let deliverSuccess = '';

  onMount(load);
  async function load() {
    loading = true;
    try { orders = await api.adminOrders(filterStatus === 'all' ? '' : filterStatus); }
    finally { loading = false; }
  }
  $: if (filterStatus !== undefined) load();

  async function setStatus(order, status) {
    try {
      const updated = await api.updateOrderStatus(order.id, status);
      orders = orders.map(o => o.id === order.id ? { ...o, status: updated.status } : o);
    } catch(e) { alert('Gagal: ' + e.message); }
  }

  function openDeliver(order) {
    deliverModal = order;
    deliverItems = '';
    deliverNote = '';
    deliverError = '';
    deliverSuccess = '';
  }

  async function doDeliver() {
    delivering = true;
    deliverError = '';
    deliverSuccess = '';
    try {
      const body = { note: deliverNote, run_script: true };
      if (deliverItems.trim() && deliverModal.product_type === 'stock') {
        body.items = deliverItems.split('\n').map(l=>l.trim()).filter(Boolean);
      }
      const res = await api.manualDeliver(deliverModal.id, body);
      deliverSuccess = res.message || 'Berhasil!';
      // Update status di list
      orders = orders.map(o => o.id === deliverModal.id
        ? {...o, status: res.status || 'paid'} : o);
      setTimeout(() => { deliverModal = null; load(); }, 1500);
    } catch(e) {
      deliverError = e.message;
    } finally {
      delivering = false;
    }
  }

  const statuses = ['pending','paid','script_executed','cancelled'];
</script>

<svelte:head><title>Pesanan — DigiStore Admin</title></svelte:head>

<div class="page-header">
  <h1 class="page-title">Daftar Pesanan</h1>
</div>

<div style="display:flex;gap:8px;margin-bottom:16px;flex-wrap:wrap">
  {#each [['all','Semua'],['paid','Lunas'],['pending','Pending'],['script_executed','Script Run'],['cancelled','Dibatalkan']] as [v,l]}
    <button class="btn btn-sm {filterStatus===v?'btn-primary':''}" on:click={()=>filterStatus=v}>{l}</button>
  {/each}
</div>

{#if loading}
  <div style="color:var(--text-muted);padding:2rem">Memuat…</div>
{:else}
  <div class="card" style="padding:0;overflow:hidden">
    <div style="overflow-x:auto">
      <table class="data-table">
        <thead>
          <tr>
            <th>Invoice</th><th>Produk</th><th>Pembeli</th><th>Total</th>
            <th>Tipe</th><th>Status</th><th>Tanggal</th><th></th>
          </tr>
        </thead>
        <tbody>
          {#each orders as o (o.id)}
            <tr>
              <td class="mono" style="font-size:11px">{o.invoice_no}</td>
              <td style="max-width:160px;white-space:nowrap;overflow:hidden;text-overflow:ellipsis">{o.product_name}</td>
              <td>
                <div style="font-size:13px">{o.buyer_name}</div>
                <div style="font-size:11px;color:var(--text-muted)">{o.buyer_email}</div>
              </td>
              <td style="font-weight:500">{IDR(o.total)}</td>
              <td><span class="badge badge-{o.product_type}">{o.product_type==='stock'?'Stok':'Script'}</span></td>
              <td>
                <select class="status-select badge-{o.status}"
                  value={o.status}
                  on:change={(e)=>setStatus(o,e.target.value)}>
                  {#each statuses as s}
                    <option value={s}>{STATUS_LABEL[s]||s}</option>
                  {/each}
                </select>
              </td>
              <td style="font-size:11.5px;color:var(--text-muted);white-space:nowrap">{fmtDateTime(o.created_at)}</td>
              <td>
                <div style="display:flex;gap:5px">
                  <button class="btn btn-sm" on:click={()=>viewOrder=o}>Invoice</button>
                  {#if o.status === 'pending'}
                    <button class="btn btn-sm" style="background:#EAF3DE;color:#2f5e0f;border-color:#c0dda8"
                      on:click={()=>openDeliver(o)}>
                      📦 Kirim
                    </button>
                  {/if}
                </div>
              </td>
            </tr>
          {/each}
        </tbody>
      </table>
    </div>
    {#if orders.length===0}
      <div style="text-align:center;padding:2.5rem;color:var(--text-muted);font-size:13px">Tidak ada pesanan.</div>
    {/if}
  </div>
{/if}

{#if viewOrder}
  <InvoiceModal order={viewOrder} on:close={()=>viewOrder=null} />
{/if}

<!-- Manual Deliver Modal -->
{#if deliverModal}
  <div class="modal-overlay" on:click={(e)=>e.target===e.currentTarget&&(deliverModal=null)} role="dialog">
    <div class="modal-box" style="max-width:520px">
      <div class="modal-header">
        <div>
          <div class="modal-title">📦 Kirim Produk Manual</div>
          <div style="font-size:12px;color:var(--text-muted);margin-top:2px">{deliverModal.invoice_no} — {deliverModal.buyer_name}</div>
        </div>
        <button class="modal-close" on:click={()=>deliverModal=null}>×</button>
      </div>

      <div style="display:flex;flex-direction:column;gap:14px">
        <!-- Info produk -->
        <div style="background:#f8f8f6;border-radius:var(--radius);padding:12px 14px;font-size:13px">
          <div><strong>{deliverModal.product_name}</strong> × {deliverModal.qty}</div>
          <div style="color:var(--text-muted);font-size:12px;margin-top:2px">
            Tipe: {deliverModal.product_type==='stock'?'Stok':'Script'} · Total: {IDR(deliverModal.total)}
          </div>
        </div>

        {#if deliverModal.product_type === 'stock'}
          <div>
            <label class="field-label">
              Item yang Dikirim <span style="color:var(--text-muted);font-weight:400">(satu per baris — opsional: kosongkan untuk ambil dari stok otomatis)</span>
            </label>
            <textarea class="input mono" rows="5" style="resize:vertical" bind:value={deliverItems}
              placeholder="Kosongkan = ambil otomatis dari stok tersedia&#10;&#10;Atau isi manual:&#10;https://drive.google.com/file/d/AAA/view&#10;LIC-KEY-XXXX-1111"></textarea>
          </div>

          <div style="background:#E6F1FB;border-radius:var(--radius);padding:10px 14px;font-size:12.5px;color:#185FA5">
            💡 Jika dikosongkan, sistem otomatis mengambil item dari tabel stok produk ini (yang belum terjual). Isi manual hanya jika ingin kirim item spesifik di luar stok terdaftar.
          </div>
        {:else}
          <div style="background:#EEEDFE;border-radius:var(--radius);padding:10px 14px;font-size:12.5px;color:#534AB7">
            ⚙️ Klik Kirim untuk menjalankan ulang semua provider actions yang dikonfigurasi pada produk ini (email, webhook, Slack, dll).
          </div>
        {/if}

        <div>
          <label class="field-label">Catatan (opsional)</label>
          <input class="input" bind:value={deliverNote} placeholder="mis: Kirim ulang karena link expired sebelumnya"/>
        </div>

        {#if deliverError}<div class="alert-error">{deliverError}</div>{/if}
        {#if deliverSuccess}<div class="alert-success">{deliverSuccess}</div>{/if}

        <div style="display:flex;gap:10px;justify-content:flex-end;border-top:0.5px solid var(--border);padding-top:14px">
          <button class="btn" on:click={()=>deliverModal=null}>Batal</button>
          <button class="btn btn-success" on:click={doDeliver} disabled={delivering}>
            {delivering ? 'Mengirim…' : '📦 Kirim Sekarang'}
          </button>
        </div>
      </div>
    </div>
  </div>
{/if}

<style>
.status-select {
  border:none;cursor:pointer;padding:3px 9px;border-radius:999px;
  font-size:11.5px;font-weight:500;font-family:inherit;outline:none;appearance:none;
}
.status-select.badge-paid    { background:#EAF3DE;color:#3B6D11; }
.status-select.badge-pending { background:#FAEEDA;color:#854F0B; }
.status-select.badge-script_executed { background:#E6F1FB;color:#185FA5; }
.status-select.badge-cancelled { background:#FCEBEB;color:#8c2626; }
</style>

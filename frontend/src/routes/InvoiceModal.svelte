<script>
  import { createEventDispatcher } from 'svelte';
  import { IDR, fmtDate, STATUS_LABEL, PAY_LABEL } from '$lib/utils.js';

  export let order;
  const dispatch = createEventDispatcher();

  function overlay(e) { if (e.target === e.currentTarget) dispatch('close'); }
</script>

<div class="modal-overlay" on:click={overlay} role="dialog" aria-modal="true">
  <div class="modal-box" style="max-width:480px">
    <div class="modal-header">
      <span class="modal-title">Invoice Pembelian</span>
      <button class="modal-close" on:click={() => dispatch('close')}>×</button>
    </div>

    <div class="success-banner">
      <div style="font-size:36px">✅</div>
      <div style="font-weight:500;font-size:15.5px;margin-top:8px">Pesanan Berhasil Dibuat!</div>
      <div style="font-size:12.5px;color:var(--text-muted);margin-top:4px">
        Invoice telah dikirim ke {order.buyer_email}
      </div>
    </div>

    <div class="invoice-box">
      <div class="inv-head">
        <div>
          <div style="font-weight:500;font-size:15px">Digital Murah</div>
          <div style="font-size:12px;color:var(--text-muted)">Produk Digital Indonesia</div>
        </div>
        <div style="text-align:right">
          <div class="mono">{order.invoice_no}</div>
          <div style="font-size:12px;color:var(--text-muted)">{fmtDate(order.created_at)}</div>
        </div>
      </div>

      <table class="inv-table">
        <tbody>
          {#each [
            ['Pembeli', order.buyer_name],
            ['Email', order.buyer_email],
            ['Produk', order.product_name],
            ['Jumlah', `${order.qty} pcs`],
            ['Harga Satuan', IDR(order.price)],
            ['Metode Bayar', PAY_LABEL[order.pay_method] || order.pay_method],
          ] as [l, v]}
            <tr>
              <td class="l">{l}</td>
              <td class="v">{v}</td>
            </tr>
          {/each}
          <tr class="total-row">
            <td>Total</td>
            <td style="color:#0d5fa8;font-size:18px">{IDR(order.total)}</td>
          </tr>
        </tbody>
      </table>

      <div class="status-row">
        <span style="font-size:13px;color:var(--text-muted)">Status</span>
        <span class="badge badge-{order.status}">{STATUS_LABEL[order.status] || order.status}</span>
      </div>
    </div>

    <div style="font-size:12px;color:var(--text-muted);text-align:center;margin-top:14px">
      Simpan nomor invoice <span class="mono" style="font-weight:500">{order.invoice_no}</span> untuk mengecek pesanan.
    </div>
  </div>
</div>

<style>
.success-banner {
  text-align: center; padding: 1.25rem;
  background: #f8f8f6; border-radius: var(--radius);
  margin-bottom: 16px;
}
.invoice-box {
  border: 0.5px solid var(--border);
  border-radius: var(--radius-lg); overflow: hidden;
}
.inv-head {
  display: flex; justify-content: space-between; align-items: flex-start;
  padding: 1rem 1.25rem;
  border-bottom: 0.5px solid var(--border);
}
.inv-table { width: 100%; border-collapse: collapse; }
.inv-table tr { border-bottom: 0.5px solid var(--border); }
.inv-table td { padding: 8px 16px; font-size: 13px; }
.inv-table td.l { color: var(--text-muted); width: 40%; }
.inv-table td.v { text-align: right; font-weight: 400; }
.total-row td { font-weight: 500; font-size: 15px; padding: 12px 16px; border-top: 2px solid #0d5fa8; }
.status-row {
  display: flex; justify-content: space-between; align-items: center;
  padding: 10px 16px; background: #f8f8f6;
}
</style>

<script>
  import { onMount } from 'svelte';
  import { api } from '$lib/api.js';
  import { fmtDateTime } from '$lib/utils.js';

  let logs = [];
  let loading = true;

  onMount(async () => {
    try { logs = await api.scriptLogs(); }
    finally { loading = false; }
  });

  function parseOutput(output) {
    try {
      const arr = JSON.parse(output);
      if (Array.isArray(arr) && arr[0]?.provider) return arr;
    } catch {}
    // Fallback: output lama berupa teks
    return output.split('\n').filter(Boolean).map(line => ({
      provider: 'log', label: 'output', enabled: true,
      status: line.includes('GAGAL') ? 'failed' : 'ok',
      output: line
    }));
  }

  const PROV_STYLE = {
    email:   { icon:'✉️',  bg:'var(--info-bg)', color:'var(--info-fg)' },
    slack:   { icon:'💬',  bg:'var(--warning-bg)', color:'var(--warning-fg)' },
    discord: { icon:'🎮',  bg:'var(--primary-bg)', color:'var(--primary)' },
    webhook: { icon:'🔗',  bg:'var(--warning-bg)', color:'var(--warning-fg)' },
    log:     { icon:'📋',  bg:'var(--success-bg)', color:'var(--success-fg)' },
  };

  const STATUS_STYLE = {
    ok:      { label:'✓ OK',      bg:'var(--success-bg)', color:'var(--success-fg)' },
    failed:  { label:'✗ Gagal',   bg:'var(--danger-bg)', color:'var(--danger-fg)' },
    skipped: { label:'— Skip',    bg:'var(--surface-2)', color:'var(--text-muted)'    },
  };
</script>

<svelte:head><title>Script Logs — Digital Murah Admin</title></svelte:head>

<div class="page-header">
  <h1 class="page-title">Script Execution Log</h1>
</div>
<p style="color:var(--text-muted);font-size:13px;margin-bottom:1.25rem">
  Riwayat eksekusi provider actions untuk setiap order produk tipe Script.
  Action dengan status <strong>Skip</strong> sengaja dinonaktifkan di pengaturan produk.
</p>

{#if loading}
  <div style="color:var(--text-muted);padding:2rem">Memuat log…</div>
{:else if logs.length === 0}
  <div class="card" style="text-align:center;padding:2.5rem;color:var(--text-muted)">Belum ada script yang dieksekusi.</div>
{:else}
  <div style="display:flex;flex-direction:column;gap:12px">
    {#each logs as log (log.id)}
      {@const actions = parseOutput(log.output)}
      {@const okCount = actions.filter(a=>a.status==='ok').length}
      {@const failCount = actions.filter(a=>a.status==='failed').length}
      {@const skipCount = actions.filter(a=>a.status==='skipped').length}

      <div class="card">
        <!-- Header -->
        <div style="display:flex;flex-wrap:wrap;align-items:center;gap:8px;margin-bottom:12px">
          <span class="badge badge-{log.status==='success'?'success':log.status==='partial'?'pending':'failed'}">
            {log.status==='success'?'✓ Sukses':log.status==='partial'?'⚡ Sebagian':'✗ Gagal'}
          </span>
          <span class="mono" style="font-size:12px">{log.invoice_no}</span>
          <span style="font-size:12px;color:var(--text-muted)">{fmtDateTime(log.created_at)}</span>
          <span style="margin-left:auto;font-size:12px;color:var(--text-muted)">
            {okCount > 0 ? `${okCount} OK` : ''} {failCount > 0 ? `· ${failCount} gagal` : ''} {skipCount > 0 ? `· ${skipCount} skip` : ''}
          </span>
        </div>
        <div style="font-weight:500;font-size:14px;margin-bottom:12px">{log.product}</div>

        <!-- Action results -->
        <div style="display:flex;flex-direction:column;gap:8px">
          {#each actions as action, i}
            {@const ps = PROV_STYLE[action.provider] || PROV_STYLE.log}
            {@const ss = STATUS_STYLE[action.status] || STATUS_STYLE.ok}
            <div style="display:flex;gap:10px;align-items:flex-start;padding:10px 12px;background:var(--surface-2);border-radius:var(--radius);border-left:3px solid {action.status==='failed'?'var(--danger-fg)':action.status==='skipped'?'var(--border-md)':'var(--success-fg)'}">
              <!-- Step number -->
              <div style="width:20px;height:20px;border-radius:50%;background:{ps.bg};color:{ps.color};font-size:11px;font-weight:600;display:flex;align-items:center;justify-content:center;flex-shrink:0;margin-top:1px">{i+1}</div>

              <div style="flex:1;min-width:0">
                <div style="display:flex;align-items:center;gap:6px;margin-bottom:5px;flex-wrap:wrap">
                  <span style="background:{ps.bg};color:{ps.color};padding:2px 8px;border-radius:999px;font-size:11.5px;font-weight:500">{ps.icon} {action.provider}</span>
                  {#if action.label}
                    <span style="font-size:12px;color:var(--text-muted)">{action.label}</span>
                  {/if}
                  <span style="background:{ss.bg};color:{ss.color};padding:2px 8px;border-radius:999px;font-size:11px;font-weight:500;margin-left:auto">{ss.label}</span>
                </div>

                {#if action.status === 'skipped'}
                  <div style="font-size:12px;color:var(--text-muted);font-style:italic">
                    Action ini dinonaktifkan — tidak dieksekusi.
                    Aktifkan kembali di pengaturan produk.
                  </div>
                {:else}
                  <div style="font-size:12px;color:{action.status==='failed'?'var(--danger-fg)':'var(--text-muted)'};word-break:break-all;font-family:'JetBrains Mono',monospace;line-height:1.6">
                    {action.output?.replace(/^\[.*?\]\s*/, '') || '—'}
                  </div>
                {/if}
              </div>
            </div>
          {/each}
        </div>
      </div>
    {/each}
  </div>
{/if}

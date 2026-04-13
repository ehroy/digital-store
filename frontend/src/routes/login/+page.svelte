<script>
  import { api, authToken } from '$lib/api.js';
  import { goto } from '$app/navigation';
  import { get } from 'svelte/store';

  // Redirect if already logged in
  import { onMount } from 'svelte';
  onMount(() => { if (get(authToken)) goto('/admin'); });

  let username = '';
  let password = '';
  let error = '';
  let loading = false;

  async function login() {
    if (!username || !password) { error = 'Username dan password wajib diisi.'; return; }
    loading = true; error = '';
    try {
      const res = await api.login(username, password);
      authToken.set(res.token);
      goto('/admin');
    } catch (e) {
      error = e.message;
    } finally {
      loading = false;
    }
  }

  function handleKey(e) { if (e.key === 'Enter') login(); }
</script>

<svelte:head><title>Login Admin — Digitalkuh Murah</title></svelte:head>

<div class="login-wrap">
  <div class="login-card">
    <div class="login-brand">
      <span class="login-logo">🛍</span>
      <span>Digitalkuh Murah Admin</span>
    </div>
    <h2 class="login-title">Masuk ke Panel Admin</h2>

    <div class="form-grid">
      <div>
        <label class="field-label">Username</label>
        <input class="input" placeholder="admin" bind:value={username} on:keydown={handleKey} />
      </div>
      <div>
        <label class="field-label">Password</label>
        <input class="input" type="password" placeholder="••••••••" bind:value={password} on:keydown={handleKey} />
      </div>

      {#if error}<div class="alert-error">{error}</div>{/if}

      <button class="btn btn-primary" style="width:100%;padding:11px;font-size:14px"
        on:click={login} disabled={loading}>
        {loading ? 'Memuat…' : 'Masuk'}
      </button>

      <a href="/" style="text-align:center;font-size:13px;color:var(--text-muted)">← Kembali ke Toko</a>
    </div>
  </div>
</div>

<style>
.login-wrap {
  min-height: 100vh;
  display: flex; align-items: center; justify-content: center;
  background: linear-gradient(135deg, #f0f6fd 0%, var(--bg) 100%);
  padding: 1rem;
}
.login-card {
  background: #fff; border-radius: var(--radius-lg);
  border: 0.5px solid var(--border);
  padding: 2rem; width: 100%; max-width: 380px;
  box-shadow: 0 4px 24px rgba(0,0,0,0.07);
}
.login-brand {
  display: flex; align-items: center; gap: 8px;
  font-weight: 500; font-size: 15px; margin-bottom: 1.5rem;
}
.login-logo {
  background: #0d5fa8; border-radius: 8px;
  width: 28px; height: 28px;
  display: flex; align-items: center; justify-content: center; font-size: 14px;
}
.login-title {
  font-size: 20px; font-weight: 500; letter-spacing: -0.3px;
  margin-bottom: 1.5rem;
}
</style>

<script>
  import { onMount } from 'svelte';

  export let floating = false;

  let theme = 'light';

  function syncTheme() {
    theme = document.documentElement.dataset.theme || 'light';
  }

  function toggleTheme() {
    const next = theme === 'dark' ? 'light' : 'dark';
    theme = next;
    document.documentElement.dataset.theme = next;
    localStorage.setItem('theme', next);
  }

  onMount(syncTheme);
</script>

<button class:floating class="btn btn-sm theme-nav-btn nav-action-btn" type="button" on:click={toggleTheme} aria-label="Toggle tema">
  {theme === 'dark' ? '☀️ Light' : '🌙 Dark'}
</button>

<style>
  .floating {
    position: fixed;
    right: 14px;
    top: 92px;
    z-index: 180;
    min-width: 88px;
    box-shadow: var(--shadow-lg);
    border-radius: 999px;
    background: var(--surface);
    border: 1px solid var(--border);
    backdrop-filter: blur(12px);
  }

  @media (max-width: 640px) {
    .floating {
      right: 8px;
      top: 65px;
      min-width: 78px;
      padding-inline: 10px;
      font-size: 11.5px;
    }
  }
</style>

<script>
  import { onMount } from 'svelte';

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

<button class="btn btn-sm theme-nav-btn" type="button" on:click={toggleTheme} aria-label="Toggle tema">
  {theme === 'dark' ? '☀️ Light' : '🌙 Dark'}
</button>

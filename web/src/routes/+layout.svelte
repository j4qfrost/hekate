<script lang="ts">
  let { children } = $props();
</script>

<header>
  <a href="/" class="brand">hekate</a>
  <nav>
    <a href="/me">me</a>
  </nav>
</header>

<main>
  {@render children()}
</main>

<footer>
  <small>
    <a href="https://github.com/hekate-events/hekate">github</a> ·
    venue-first events on the AT Protocol
  </small>
</footer>

<style>
  :global(html, body) {
    margin: 0;
    font-family: system-ui, -apple-system, sans-serif;
    background: #0b132b;
    color: #e0e6f0;
  }
  header {
    display: flex;
    justify-content: space-between;
    align-items: center;
    padding: 1rem 2rem;
    border-bottom: 1px solid #1c2541;
  }
  .brand {
    font-weight: 700;
    font-size: 1.25rem;
    color: #e0e6f0;
    text-decoration: none;
  }
  nav a {
    color: #8aa1d1;
    margin-left: 1rem;
    text-decoration: none;
  }
  nav a:hover {
    color: #e0e6f0;
  }
  main {
    max-width: 60rem;
    margin: 0 auto;
    padding: 2rem;
  }
  footer {
    text-align: center;
    padding: 2rem;
    color: #5d769a;
  }
  footer a {
    color: #8aa1d1;
  }
</style>

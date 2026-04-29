<script lang="ts">
  import { onMount } from 'svelte';
  import { hekate } from '$lib/api';

  let serverOk = $state<boolean | null>(null);

  onMount(async () => {
    try {
      serverOk = await hekate.healthz();
    } catch {
      serverOk = false;
    }
  });
</script>

<svelte:head>
  <title>hekate — venue-first events on the AT Protocol</title>
</svelte:head>

<section class="hero">
  <h1>Venues you can find. Slots you can claim.</h1>
  <p>
    hekate is a federated event-coordination protocol on the AT Protocol. Venues
    publish discrete time-slots and recurring schedules; organizers claim a slot
    by publishing an event.
  </p>
  <p>
    All state lives in your PDS. This indexer is read-only — anyone can run their
    own.
  </p>
</section>

<section class="status">
  <h2>Indexer status</h2>
  {#if serverOk === null}
    <p>checking…</p>
  {:else if serverOk}
    <p class="ok">indexer up · ready for queries (queries land with M1)</p>
  {:else}
    <p class="err">indexer unreachable — see <code>docs/SELFHOST.md</code> for bring-up</p>
  {/if}
</section>

<section>
  <h2>Roadmap snapshot</h2>
  <ol>
    <li><strong>M-1</strong> Smoke Signal coordination (week 1)</li>
    <li><strong>M0</strong> Lexicon v0.1 (by 2026-05-29)</li>
    <li><strong>M1</strong> Server v0.1 (by 2026-10-29 — kill date)</li>
    <li><strong>M2</strong> CLI v0.1 — the v0.1 reference client</li>
    <li><strong>M3</strong> Web v0.1 (you are here, scaffolded only)</li>
  </ol>
  <p><a href="https://github.com/hekate-events/hekate/blob/main/docs/ROADMAP.md">Full roadmap →</a></p>
</section>

<style>
  .hero h1 {
    font-size: 2.5rem;
    margin: 0 0 1rem;
  }
  .hero p {
    color: #b0bcd1;
    max-width: 36rem;
  }
  .status {
    margin-top: 3rem;
  }
  .ok {
    color: #6ee7b7;
  }
  .err {
    color: #fca5a5;
  }
  ol li {
    margin: 0.4rem 0;
  }
</style>

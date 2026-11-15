<script lang="ts">
  import Nav from "./lib/Nav.svelte";
  import Overview from "./pages/Overview.svelte";
  import Sql from "./pages/Sql.svelte";

  const read = () => (location.hash.replace(/^#\/?/, "") || "overview").split("?")[0];
  let page = $state(read());
  $effect(() => {
    const on = () => (page = read());
    window.addEventListener("hashchange", on);
    return () => window.removeEventListener("hashchange", on);
  });
</script>

<div class="app">
  <Nav {page} />
  <main>
    {#if page === "sql"}
      <Sql />
    {:else}
      <Overview />
    {/if}
  </main>
</div>

<style>
  .app {
    display: flex;
    min-height: 100vh;
  }
  main {
    flex: 1;
    min-width: 0;
  }
</style>

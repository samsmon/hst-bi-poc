<script lang="ts">
  type Status = "checking" | "ok" | "down";
  let server = $state<Status>("checking");

  $effect(() => {
    fetch("/health")
      .then((r) => (server = r.ok ? "ok" : "down"))
      .catch(() => (server = "down"));
  });
</script>

<main>
  <h1>hst-bi</h1>
  <p class="muted">
    Foundation only. Dashboards and the SQL console arrive in phase P4.
  </p>
  <p>Server: {server}</p>
</main>

<style>
  main {
    max-width: 720px;
    margin: 4rem auto;
    padding: 0 1rem;
  }
  .muted {
    color: var(--muted);
  }
</style>

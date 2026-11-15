<script lang="ts">
  let { page }: { page: string } = $props();
  const items = [
    { id: "overview", label: "Overview" },
    { id: "sql", label: "SQL console" },
  ];
</script>

<nav>
  <div class="brand">
    <strong>hst-bi</strong>
    <span class="caption">HST ticket sales</span>
  </div>
  <p class="caption group">Dashboards</p>
  <a href="#/overview" class:selected={page === "overview"} aria-current={page === "overview" ? "page" : undefined}>Overview</a>
  <p class="caption group">Analysis</p>
  <a href="#/sql" class:selected={page === "sql"} aria-current={page === "sql" ? "page" : undefined}>SQL console</a>
  <p class="caption foot">Pages for demand, routes, channels, refunds and forecasts come next.</p>
</nav>

<style>
  nav {
    width: 208px;
    flex: none;
    padding: 16px;
    border-right: 1px solid var(--line);
    display: flex;
    flex-direction: column;
    gap: 2px;
  }
  .brand {
    display: flex;
    flex-direction: column;
    margin-bottom: 4px;
  }
  .group {
    margin: 14px 0 2px;
  }
  a {
    color: var(--ink);
    text-decoration: none;
    padding: 8px 12px;
    border-radius: 4px;
  }
  a:hover {
    background: var(--accent-soft);
  }
  a.selected {
    background: var(--accent-soft);
    font-weight: 500;
  }
  .foot {
    margin-top: auto;
  }
</style>

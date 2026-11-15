<script lang="ts">
  import { query, type Failure, type Result } from "../lib/api";
  import { byFormat, integer, percent } from "../lib/format";
  import SeasonBars from "../lib/SeasonBars.svelte";

  let kpi = $state<Result | null>(null);
  let trend = $state<Result | null>(null);
  let channels = $state<Result | null>(null);
  let routes = $state<Result | null>(null);
  let error = $state<string | null>(null);
  let loading = $state(true);

  async function load() {
    loading = true;
    error = null;
    try {
      [kpi, trend, channels, routes] = await Promise.all([
        query({ measures: ["tickets", "gross_revenue", "net_revenue", "refund_rate", "avg_lead_time"] }),
        query({ measures: ["net_revenue"], dimensions: ["season_type"], grain: "month", limit: 5000 }),
        query({ measures: ["tickets"], dimensions: ["channel"], order: [{ by: "tickets", dir: "desc" }] }),
        query({ measures: ["tickets", "net_revenue", "refund_rate"], dimensions: ["od"], order: [{ by: "tickets", dir: "desc" }], limit: 5 }),
      ]);
    } catch (e) {
      error = (e as Failure).detail;
    } finally {
      loading = false;
    }
  }
  $effect(() => {
    load();
  });

  const tiles = [
    { label: "Tickets", col: 0, fmt: "integer" },
    { label: "Gross revenue", col: 1, fmt: "idr" },
    { label: "Net revenue", col: 2, fmt: "idr" },
    { label: "Refund rate", col: 3, fmt: "percent" },
    { label: "Average lead time", col: 4, fmt: "days" },
  ];
  const channelTotal = $derived(channels ? channels.rows.reduce((s, r) => s + (r[1] as number), 0) : 1);
  const channelMax = $derived(channels ? Math.max(...channels.rows.map((r) => r[1] as number)) : 1);
</script>

<header>
  <h1>Overview</h1>
  <span class="caption">All years, 2020 to 2025. Date and segment filters arrive with the next iteration.</span>
</header>

<div class="content">
  {#if error}
    <section class="plate">
      <h2>The data did not load</h2>
      <p class="error">{error}</p>
      <button onclick={load}>Retry</button>
    </section>
  {:else}
    <section class="kpis" aria-busy={loading}>
      {#each tiles as t}
        <div class="tile">
          <span class="caption">{t.label}</span>
          <strong class="num">{kpi ? byFormat(t.fmt, kpi.rows[0]?.[t.col]) : " "}</strong>
        </div>
      {/each}
    </section>

    <section class="plate">
      <div class="head">
        <h2>Net revenue by month and season type</h2>
        <span class="caption">IDR billions, stacked by season type</span>
      </div>
      {#if trend}<SeasonBars rows={trend.rows} />{:else}<div class="skeleton" style="height: 240px"></div>{/if}
    </section>

    <div class="row">
      <section class="plate">
        <div class="head"><h2>Channel mix</h2><span class="caption">Share of tickets</span></div>
        {#if channels}
          {#each channels.rows as r}
            <div class="bar-row">
              <span>{r[0]}</span>
              <div class="track"><div class="bar" style="width: {((r[1] as number) / channelMax) * 100}%"></div></div>
              <span class="num">{percent((r[1] as number) / channelTotal, 1)}</span>
            </div>
          {/each}
        {:else}<div class="skeleton" style="height: 160px"></div>{/if}
      </section>

      <section class="plate">
        <div class="head"><h2>Top routes</h2><span class="caption">By tickets</span></div>
        {#if routes}
          <table>
            <thead><tr><th>Route</th><th class="num">Tickets</th><th class="num">Net revenue</th><th class="num">Refund rate</th></tr></thead>
            <tbody>
              {#each routes.rows as r}
                <tr><td>{r[0]}</td><td class="num">{integer(r[1])}</td><td class="num">{byFormat("idr", r[2])}</td><td class="num">{percent(r[3])}</td></tr>
              {/each}
            </tbody>
          </table>
        {:else}<div class="skeleton" style="height: 160px"></div>{/if}
      </section>
    </div>

    <p class="caption">
      Source: bi.mv_monthly_sales{trend ? ` | queries ${[kpi, trend, channels, routes].map((r) => r?.duration_ms ?? 0).reduce((a, b) => a + b, 0)} ms in total` : ""}
    </p>
  {/if}
</div>

<style>
  header {
    padding: 14px 24px;
    border-bottom: 1px solid var(--line);
    display: flex;
    align-items: baseline;
    gap: 16px;
  }
  .content {
    padding: 24px;
    display: flex;
    flex-direction: column;
    gap: 16px;
  }
  .kpis {
    display: grid;
    grid-template-columns: repeat(5, 1fr);
  }
  .tile {
    background: var(--surface);
    border: 1px solid var(--line);
    margin-left: -1px;
    padding: 16px;
    display: flex;
    flex-direction: column;
    gap: 4px;
  }
  .tile strong {
    font-size: 28px;
    line-height: 36px;
    font-weight: 600;
    text-align: left;
  }
  .head {
    display: flex;
    justify-content: space-between;
    align-items: baseline;
    margin-bottom: 12px;
  }
  .row {
    display: grid;
    grid-template-columns: 1fr 1fr;
    gap: 16px;
  }
  .bar-row {
    display: grid;
    grid-template-columns: 80px 1fr 56px;
    gap: 12px;
    align-items: center;
    padding: 5px 0;
  }
  .track {
    height: 16px;
  }
  .bar {
    height: 16px;
    background: var(--s1);
  }
  table {
    width: 100%;
    border-collapse: collapse;
  }
  th {
    font-size: 12px;
    font-weight: 400;
    color: var(--ink-muted);
    text-align: left;
    padding: 6px 0;
    border-bottom: 1px solid var(--line);
  }
  td {
    padding: 9px 0;
    border-bottom: 1px solid var(--line);
  }
  .skeleton {
    background: var(--line);
    opacity: 0.4;
  }
  @media (max-width: 1100px) {
    .kpis { grid-template-columns: repeat(2, 1fr); }
    .row { grid-template-columns: 1fr; }
  }
</style>

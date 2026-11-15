<script lang="ts">
  import { runSql, schema, type Failure, type Result, type Table } from "../lib/api";
  import { cell, isNumber } from "../lib/format";

  const starter = `SELECT season_type,
       date_trunc('month', travel_date)::date AS month,
       sum(net_revenue) AS net_revenue
FROM public.fact_ticket_sales_wide
WHERE travel_date >= DATE '2025-01-01'
GROUP BY 1, 2
ORDER BY 2, 1;`;

  let text = $state(starter);
  let result = $state<Result | null>(null);
  let failure = $state<Failure | null>(null);
  let running = $state(false);
  let tables = $state<Table[]>([]);
  let schemaError = $state<string | null>(null);
  let open = $state<Record<string, boolean>>({});
  let filter = $state("");
  let editor: HTMLTextAreaElement | undefined = $state();
  let controller: AbortController | null = null;

  $effect(() => {
    schema()
      .then((s) => {
        tables = s.tables;
        const first = s.tables.find((t) => t.name === "fact_ticket_sales_wide");
        if (first) open[`${first.schema}.${first.name}`] = true;
      })
      .catch((e: Failure) => (schemaError = e.detail));
  });

  async function run() {
    if (running) return;
    const sel = editor && editor.selectionStart !== editor.selectionEnd ? text.slice(editor.selectionStart, editor.selectionEnd) : text;
    running = true;
    failure = null;
    controller = new AbortController();
    try {
      result = await runSql(sel);
    } catch (e) {
      failure = e as Failure;
      result = null;
    } finally {
      running = false;
      controller = null;
    }
  }
  function onKey(e: KeyboardEvent) {
    if ((e.ctrlKey || e.metaKey) && e.key === "Enter") {
      e.preventDefault();
      run();
    }
  }
  function insert(name: string) {
    if (!editor) return;
    const a = editor.selectionStart;
    text = text.slice(0, a) + name + text.slice(editor.selectionEnd);
    queueMicrotask(() => {
      editor!.focus();
      editor!.setSelectionRange(a + name.length, a + name.length);
    });
  }
  function toCsv(r: Result) {
    const esc = (v: unknown) => {
      const s = v === null || v === undefined ? "" : String(v);
      return /[",\n]/.test(s) ? `"${s.replaceAll('"', '""')}"` : s;
    };
    return [r.columns.map(esc).join(","), ...r.rows.map((row) => row.map(esc).join(","))].join("\n");
  }
  function exportCsv() {
    if (!result) return;
    const url = URL.createObjectURL(new Blob([toCsv(result)], { type: "text/csv" }));
    const a = Object.assign(document.createElement("a"), { href: url, download: "result.csv" });
    a.click();
    URL.revokeObjectURL(url);
  }

  const shown = $derived(
    tables.filter((t) => !filter || `${t.schema}.${t.name}`.includes(filter) || t.columns.some((c) => c.name.includes(filter))),
  );
</script>

<header>
  <h1>SQL console</h1>
  <span class="caption">Read-only. One SELECT per run, 10,000 rows, 30 second limit.</span>
</header>

<div class="layout">
  <aside class="plate">
    <input type="search" placeholder="Search tables and columns" bind:value={filter} aria-label="Search tables and columns" />
    {#if schemaError}<p class="error caption">{schemaError}</p>{/if}
    {#each shown as t (t.schema + "." + t.name)}
      {@const id = `${t.schema}.${t.name}`}
      <button class="tree" onclick={() => (open[id] = !open[id])} aria-expanded={!!open[id]}>
        <span>{open[id] ? "▾" : "▸"}</span><span>{id}</span><span class="caption">{t.kind === "table" ? "" : "view"}</span>
      </button>
      {#if open[id] || filter}
        {#each t.columns.filter((c) => !filter || c.name.includes(filter) || open[id]) as c}
          <button class="col" onclick={() => insert(c.name)} title={c.description ?? ""}>
            <code>{c.name}</code><span class="caption">{c.type}</span>
          </button>
        {/each}
      {/if}
    {/each}
  </aside>

  <section class="work">
    <div class="plate editor">
      <div class="toolbar">
        <button class="primary" onclick={run} disabled={running}>{running ? "Running..." : "Run  Ctrl+Enter"}</button>
        <span class="caption">Runs the selection, or the whole text.</span>
      </div>
      <textarea bind:this={editor} bind:value={text} onkeydown={onKey} spellcheck="false" aria-label="SQL query" rows="9"></textarea>
      <div class="status caption">
        {#if running}Running...
        {:else if failure}<span class="error">{failure.rejected ? "Rejected by the SQL guard: " : failure.timeout ? "Timeout: " : "Error: "}{failure.detail}</span>
        {:else if result}Accepted by the SQL guard | {result.rows.length} rows{result.truncated ? " (truncated at the row limit)" : ""} | {result.duration_ms} ms
        {:else}Not run yet{/if}
      </div>
    </div>

    <div class="plate results">
      <div class="toolbar">
        <h2>Results</h2>
        <span class="grow"></span>
        <button onclick={exportCsv} disabled={!result || result.rows.length === 0}>Export CSV</button>
      </div>
      {#if result}
        {#if result.truncated}<p class="caption">Showing the first {result.rows.length} rows. Narrow the query to see the rest.</p>{/if}
        <div class="scroll">
          <table>
            <thead><tr>{#each result.columns as c, i}<th class:num={result.rows.length && isNumber(result.rows[0][i])}>{c}</th>{/each}</tr></thead>
            <tbody>
              {#each result.rows as row}
                <tr>{#each row as v}<td class:num={isNumber(v)}>{cell(v)}</td>{/each}</tr>
              {/each}
            </tbody>
          </table>
        </div>
      {:else if !failure}
        <p class="muted">Run a query to see rows here.</p>
      {/if}
    </div>
  </section>
</div>

<style>
  header {
    padding: 14px 24px;
    border-bottom: 1px solid var(--line);
    display: flex;
    align-items: baseline;
    gap: 16px;
  }
  .layout {
    display: grid;
    grid-template-columns: 264px 1fr;
    gap: 16px;
    padding: 24px;
    align-items: start;
  }
  aside {
    display: flex;
    flex-direction: column;
    gap: 2px;
    max-height: calc(100vh - 120px);
    overflow: auto;
  }
  input[type="search"] {
    font: inherit;
    color: var(--ink);
    background: var(--bg);
    border: 1px solid var(--line);
    border-radius: 4px;
    padding: 6px 12px;
    margin-bottom: 8px;
  }
  button.tree,
  button.col {
    border: 0;
    background: none;
    padding: 3px 0;
    text-align: left;
    display: flex;
    gap: 6px;
    width: 100%;
  }
  button.tree span:nth-child(2) {
    flex: 1;
  }
  button.col {
    padding-left: 18px;
    justify-content: space-between;
  }
  code {
    font-family: "Ubuntu Sans Mono", ui-monospace, Consolas, monospace;
    font-size: 13px;
  }
  .work {
    display: flex;
    flex-direction: column;
    gap: 16px;
    min-width: 0;
  }
  .editor {
    padding: 0;
  }
  .toolbar {
    display: flex;
    align-items: center;
    gap: 12px;
    padding: 12px 16px;
  }
  .grow {
    flex: 1;
  }
  textarea {
    width: 100%;
    border: 0;
    border-top: 1px solid var(--line);
    background: var(--surface);
    color: var(--ink);
    padding: 16px;
    font-family: "Ubuntu Sans Mono", ui-monospace, Consolas, monospace;
    font-size: 13px;
    line-height: 20px;
    resize: vertical;
  }
  .status {
    border-top: 1px solid var(--line);
    padding: 8px 16px;
  }
  .results {
    padding: 0;
  }
  .results > p {
    padding: 0 16px 16px;
    margin: 0;
  }
  .scroll {
    overflow: auto;
    max-height: 420px;
    padding: 0 16px 16px;
  }
  table {
    border-collapse: collapse;
    width: 100%;
  }
  th {
    position: sticky;
    top: 0;
    background: var(--surface);
    font-size: 12px;
    font-weight: 400;
    color: var(--ink-muted);
    text-align: left;
    padding: 6px 12px 6px 0;
    border-bottom: 1px solid var(--line);
  }
  td {
    padding: 7px 12px 7px 0;
    border-bottom: 1px solid var(--line);
    white-space: nowrap;
  }
  th.num,
  td.num {
    text-align: right;
  }
</style>

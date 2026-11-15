<script lang="ts">
  let { rows }: { rows: unknown[][] } = $props();

  // Season types are exclusive per date, so one bar per month stacks them.
  const series: { key: string[]; label: string; color: string }[] = [
    { key: ["normal"], label: "Normal", color: "var(--s1)" },
    { key: ["school_holiday"], label: "School holiday", color: "var(--s3)" },
    { key: ["lebaran_mudik", "lebaran_balik"], label: "Lebaran (mudik and balik)", color: "var(--s5)" },
    { key: ["nataru"], label: "Nataru", color: "var(--s2)" },
    { key: ["covid_restriction"], label: "COVID restriction", color: "var(--s-other)" },
  ];
  const W = 1136;
  const H = 200;
  const LEFT = 40;

  const model = $derived.by(() => {
    const months = [...new Set(rows.map((r) => r[0] as string))].sort();
    const value = new Map<string, number>();
    for (const r of rows) value.set(`${r[0]}|${r[1]}`, (r[2] as number) / 1e9);
    const stacks = months.map((m) => {
      let acc = 0;
      return series.map((s) => {
        const v = s.key.reduce((sum, k) => sum + (value.get(`${m}|${k}`) ?? 0), 0);
        const from = acc;
        acc += v;
        return { v, from };
      });
    });
    const max = Math.max(1, ...stacks.map((s) => s.reduce((a, b) => a + b.v, 0)));
    const top = Math.ceil(max / 20) * 20;
    return { months, stacks, top };
  });

  const step = $derived((W - LEFT) / Math.max(model.months.length, 1));
  const y = (v: number) => H - (v / model.top) * H;
  const ticks = $derived([0, 1, 2, 3].map((i) => (model.top / 3) * i));
  const years = $derived(model.months.map((m, i) => ({ m, i })).filter((x) => x.m.endsWith("-01-01")));
</script>

<div class="legend">
  {#each series as s}
    <span><i style="background: {s.color}"></i>{s.label}</span>
  {/each}
</div>
<svg viewBox="0 0 {W} {H + 22}" role="img" aria-label="Net revenue by month, stacked by season type">
  {#each ticks as t}
    <line x1={LEFT} x2={W} y1={y(t)} y2={y(t)} stroke="var(--line)" />
    <text x="0" y={y(t) + 4} class="caption">{Math.round(t)}</text>
  {/each}
  {#each model.stacks as stack, i}
    {#each stack as seg, k}
      {#if seg.v > 0}
        <rect x={LEFT + i * step + (step - 11) / 2} y={y(seg.from + seg.v)} width="11" height={Math.max(1, y(seg.from) - y(seg.from + seg.v))} fill={series[k].color}>
          <title>{model.months[i]} {series[k].label}: IDR {seg.v.toFixed(1)} B</title>
        </rect>
      {/if}
    {/each}
  {/each}
  {#each years as yr}
    <text x={LEFT + yr.i * step + 2} y={H + 16} class="caption">{yr.m.slice(0, 4)}</text>
  {/each}
</svg>

<style>
  svg {
    width: 100%;
    height: auto;
    display: block;
  }
  text {
    fill: var(--ink-muted);
    font-size: 12px;
  }
  .legend {
    display: flex;
    flex-wrap: wrap;
    gap: 20px;
    margin-bottom: 8px;
    font-size: 12px;
  }
  .legend span {
    display: flex;
    align-items: center;
    gap: 6px;
  }
  .legend i {
    width: 10px;
    height: 10px;
    display: inline-block;
  }
</style>

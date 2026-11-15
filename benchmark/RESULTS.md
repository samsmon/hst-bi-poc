# Benchmark results (phase P3)

Measured on 2026-11-15 on the target host: a 4 vCPU, 8 GB container that also runs about 24 other services and the shared PostgreSQL 16. The table is `public.fact_ticket_sales_wide` (10,815,864 rows, 43 columns, 4.06 GB with indexes). All queries ran as the read-only role. Each query ran twice in a row; the first run is "run 1", the second "run 2". Nothing was run in parallel. These are single measurements on a shared machine, so treat them as indications, not as a rigorous benchmark.

Reproduce with `run_p3.sh` (and `run_evidence.sh` for the Evidence part). Queries are in `queries.tsv`.

## 1. PostgreSQL: raw table against the materialized view

`bi.mv_monthly_sales` (from `sql/03_bi_matviews.sql`) is a cube of month x season x channel x origin x destination x seat class. It took 22 seconds to build and holds 22,722 rows in 3.8 MB.

| Query | Raw, run 1 | Raw, run 2 | View, run 1 | View, run 2 |
|---|---|---|---|---|
| q1 yearly KPIs | 6,520 ms | 1,847 ms | 12 ms | 12 ms |
| q2 monthly net revenue by season | 1,980 ms | 1,976 ms | 9 ms | 8 ms |
| q3 top routes | 2,088 ms | 2,174 ms | 12 ms | 11 ms |
| q4 channel per year | 1,696 ms | 1,691 ms | 12 ms | 12 ms |
| q5 refund rate by lead-time bucket | 1,275 ms | 1,277 ms | not covered | |
| q6 occupancy per trip, 2025 | 451 ms | 507 ms | not covered | |
| q7 distinct customers per year | 9,968 ms | 9,833 ms | not covered | |
| q8 one month, filtered | 47 ms | 47 ms | 4 ms | 3 ms |

Earlier the same day, with a cold cache, q1 took 13.0 s, q2 6.7 s and q3 5.8 s on the raw table.

Reading: the view answers the dashboard queries it covers in about 10 ms, far below the 2 second target. Raw-table aggregates sit at or above the target even when warm, and much higher when cold. Distinct counts (q7) and anything keyed by lead time or trip (q5, q6) need their own pre-aggregation.

## 2. Superset 5.0.0

Official image plus the PostgreSQL driver, one gunicorn worker, SQLite metadata, no cache, no Celery.

| Measure | Value |
|---|---|
| Image size | 3.74 GB |
| Time to ready, including pull and build | 273 s |
| Memory idle | 198 MiB |
| Memory after the query run | 199 MiB |
| Login page | 200, 0.16 s, 15 KB HTML |

Queries through the SQL Lab API (synchronous), total request time:

| Query | Raw, run 1 | Raw, run 2 | View, run 1 | View, run 2 |
|---|---|---|---|---|
| q1 | 21,252 ms | 4,153 ms | 130 ms | 131 ms |
| q2 | 3,106 ms | 4,059 ms | 327 ms | 112 ms |
| q3 | 3,609 ms | 4,964 ms | 478 ms | 120 ms |
| q4 | 2,261 ms | 3,649 ms | 1,020 ms | 113 ms |
| q5 | 3,503 ms | 2,779 ms | | |
| q6 | 3,946 ms | 1,182 ms | | |
| q7 | 7,067 ms | 6,660 ms | | |
| q8 | 170 ms | 153 ms | 103 ms | 112 ms |

Reading: about 100 ms of overhead per query on top of PostgreSQL once warm. Idle memory was far lower than the 2 GB we had budgeted. Dashboard rendering in a browser was not measured.

## 3. Evidence (open-source release 40.1.8)

The current template targets the hosted Evidence Studio CLI, which needs a login, so the run pins the last open-source template (2026-02-06). The npm package was last published on that date.

| Measure | Value |
|---|---|
| Source extraction (4 aggregated queries, 114 rows) | 29 s |
| Static build | 95 s |
| Build output | 88.5 MB |
| Peak memory during build | reached the 2 GiB limit; the same build was OOM-killed at 1 GiB |
| Memory while serving (`npx serve`) | 152 MiB |
| Page response | 200, 2 to 5 ms, 14.5 KB HTML |
| `npm install` | not measured (the measured run reused a cached install) |

Reading: pages are fast because they are static, but data is frozen at build time, the build needs about 2 GB of memory for a four-query page, and there is no SQL console.

## 4. Fit against our requirements

| Requirement | Superset | Evidence (open source) | Own app (this repo) |
|---|---|---|---|
| Fully custom UI matching the mockups | Themes and CSS only; deeper changes mean patching a large codebase | Markdown plus components; custom Svelte components possible | Full control |
| SQL analysis page | Yes (SQL Lab), limited restyling | No | To be built (guard done) |
| Forecast display with bands and run compare | Possible with chart plugins, not out of the box | Possible with custom components | To be built (designed) |
| Live data and filters hitting the database | Yes | No (build-time data) | Yes |
| Idle memory | about 200 MiB | about 150 MiB serving, 2 GiB to build | 52 MiB measured on Windows; Linux not measured yet |
| Image or build size | 3.74 GB image | 88.5 MB site | about 14 MB binary |
| Maintenance outlook | Active Apache project | Open-source line looks frozen since 2026-02 | Ours to maintain |

## 5. What this means for the plan

1. Pre-aggregation is required. Keep `bi.mv_monthly_sales`, add views for distinct customers, lead-time buckets and trip occupancy, and refresh them after each load.
2. Superset is a workable fallback for ad hoc SQL and quick charts at about 200 MiB, but it cannot deliver the custom UI.
3. Evidence is not a fit: no SQL console, build-time data, heavy builds, and an open-source line that appears to have stopped.
4. The own app stays the main path. Its server still needs the Docker build and a memory measurement on Linux.

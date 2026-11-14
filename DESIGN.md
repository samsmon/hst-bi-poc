# Design

Source of truth for the UI. The Figma file implements these tokens as variables, text styles, and components; screenshots of the three mockups are in `design/`. Screen flow and states are in [WORKFLOW.md](WORKFLOW.md).

## Figma file
- File: https://www.figma.com/design/EJMq4BniLbnn3co7i1sAhy (private; the screenshots below are the shareable copy).
- Pages: `Screens` (Overview, Forecast, SQL console, each in Light and Dark, plus States) and `Components`.
- Variables: collections `Color` (Light values), `Color Dark` (same names, Dark values), and `Space` (4, 8, 12, 16, 24, 32). Names follow the token tables below, for example `color/ink`, `color/series/1`, `space/lg`. Light and Dark are two collections rather than two modes because the Figma plan in use allows one mode per collection. Rebind a frame from `Color` to `Color Dark` to switch theme; on the web both map to the same CSS custom properties.
- Text styles: Page title, Section heading, Body, Body strong, Table cell, KPI figure, Caption, SQL.
- Components: Button / Primary, Button / Secondary, Nav / Item, Nav / Item selected, Filter / Field, KPI / Tile, each with text properties.

| Screen | Light | Dark |
|---|---|---|
| Overview | [overview-light.png](design/overview-light.png) | [overview-dark.png](design/overview-dark.png) |
| Forecast | [forecast-light.png](design/forecast-light.png) | [forecast-dark.png](design/forecast-dark.png) |
| SQL console | [sql-console-light.png](design/sql-console-light.png) | [sql-console-dark.png](design/sql-console-dark.png) |
| States (loading, empty, error, rejected, truncated, stale, forbidden) | [states-light.png](design/states-light.png) | |

Numbers in the mockups are real values queried from `hst_dataset` on 2026-11-15 through the read-only role: KPIs, the monthly revenue by season type, channel mix, top routes, and the SQL console query with its 32 result rows and 1.9 s timing. On the Forecast screen the actuals are real, and the forecast is a seasonal-naive baseline computed from them (2026 month = same month of 2025), with an 80% band from the 10th to 90th percentile of its 2025 backtest errors (WAPE 3.9%, MAE 8,080 tickets). It is labeled as demo data because it is not the team's model. The States sheet uses example copy.

What the real data changed in the design:
- Season types are date-exclusive, so the Overview trend is a stacked monthly bar chart, not one line per season.
- `COVID restriction` is its own season series (gray).
- Seat capacity is not a constant 600: trips carry 300, 450, or 600 seats, so capacity is summed per trip.
- Forecast demand peaks at 68% of capacity, so the capacity panel shows "no shortfall" instead of a warning.
- Full-table aggregates take 6 to 13 seconds without pre-aggregation, far above the 2 second target; the footer shows the measured time.

## Brief
- Purpose: explore ticket sales of the HST dataset, run read-only SQL, and read ML forecasts next to actuals.
- Users: the analyst who owns the project, lecturers and reviewers who open a shared link, teammates reading forecasts.
- Tone: a precise instrument, not a marketing page. Calm, dense, legible. Closer to a timetable board than a SaaS landing page.

## Principles
1. The data is the interface. Charts and tables get the space; chrome stays quiet.
2. Honest by default: every number shows unit, filter state, and the period it covers. Forecasts never look like actuals.
3. Nothing decorative that does nothing. A control exists only if it works. No placeholder panels.
4. Dense but readable: right-aligned tabular numbers, one consistent IDR format, short labels.
5. One accent. Color encodes data first, emphasis second.

## Color tokens (60-30-10)
Roughly 60% neutral ground, 30% structure and text, 10% accent and data color.

| Token | Role | Light | Dark |
|---|---|---|---|
| `bg` | Page ground (60%) | `#F6F5F1` | `#121417` |
| `surface` | Table, editor, and chart plates; flat, no shadow | `#FFFFFF` | `#1A1D21` |
| `ink` | Primary text (30%) | `#1A1C1E` | `#E7E8E4` |
| `ink-muted` | Secondary text, axis labels | `#5E6368` | `#9AA0A6` |
| `line` | Dividers, gridlines, borders | `#D8D6CE` | `#2B2F34` |
| `accent` | Selection, focus, primary action, series 1 (10%) | `#0072B2` | `#56B4E9` |
| `accent-soft` | Selected row and brush fill | `#DCE9F3` | `#1D2B38` |
| `negative` | Drops, errors | `#D55E00` | `#E8803A` |
| `caution` | Demo data, stale forecast | `#8A5A00` text on `#F6E7C4` | `#F0B429` text on `#33290F` |

Series palette (colorblind-safe, Okabe-Ito based; never rely on color alone, always label or use a different dash or marker):

| Series | Light | Dark |
|---|---|---|
| 1 | `#0072B2` | `#56B4E9` |
| 2 | `#D55E00` | `#E8803A` |
| 3 | `#009E73` | `#2BC299` |
| 4 | `#CC79A7` | `#E39CC4` |
| 5 | `#E69F00` | `#F0B429` |
| 6 | `#56B4E9` | `#8FBCE6` |
| other | `#6B6F73` | `#9AA0A6` |

Contrast target: text at least 4.5:1 against its ground in both modes; chart marks at least 3:1.

## Typography
| Use | Font | Size / line | Weight |
|---|---|---|---|
| Page title | Inter | 24 / 32 | 600 |
| Section heading | Inter | 16 / 24 | 600 |
| Body, controls | Inter | 14 / 20 | 400 |
| Table cell | Inter, tabular numerals (`font-variant-numeric: tabular-nums` on the web) | 13 / 18 | 400 |
| KPI figure | Inter, tabular numerals | 32 / 36 | 600 |
| Caption, axis | Inter | 12 / 16 | 400, `ink-muted` |
| SQL editor | Ubuntu Sans Mono (or a system monospace) | 13 / 20 | 400 |

Contrast in type comes from size and weight steps, not from decoration. No letter-spacing tricks, no all-caps headings. Both families are open licensed and can be self-hosted.

## Space and layout
- 8 px grid. Spacing steps: 4, 8, 12, 16, 24, 32.
- Desktop frame 1440 x 900. Left navigation 208 px. Top bar 56 px with the page title and the global filter bar. Content padding 24 px, 12 columns, 16 px gutter.
- Works from 1280 px down to tablet width; phone is out of scope for the MVP.
- Corners: 4 px on controls, 0 on plates and tables. Strokes are 1 px `line`. No shadows, no gradients.

## Ergonomics
- Minimum control height 32 px (dense) and 40 px for primary actions; hit area never below 32 px.
- Visible focus ring: 2 px `accent`, 2 px offset. Full keyboard path for filters, tables, and the SQL console (`Ctrl+Enter` run, `Esc` cancel, `Ctrl+K` command palette).
- Motion (GSAP) is for orientation only: panel and page transitions 160 to 240 ms, KPI counters 400 ms, list reorder with Flip. Chart data points are not animated by GSAP. Respect `prefers-reduced-motion` by switching to instant changes.
- Filter state lives in the URL so any view can be shared and reopened.

## Visual language for predictions
| Element | Style |
|---|---|
| Actual | Solid line, 2 px |
| Forecast | Dashed line (6 4), same hue as its actual series |
| Interval | Translucent band (20% opacity) around the forecast; level in the legend, for example 80% |
| Forecast origin | Thin vertical rule labeled with the date |
| Demo data | Persistent `caution` banner while any demo run is shown |
| Model info | Model name, version, `trained_until`, `forecast_origin` printed beside the chart, never only in a tooltip |
| Staleness | Banner when the latest run is older than the agreed threshold |

## Anti-slop checklist (review every screen against this)
- [ ] No card with rounded corners and a colored accent line on the left or in a corner.
- [ ] No gradients, glass, glow, stock illustration, or emoji used as icons.
- [ ] Exactly one accent color; series colors appear only where they encode data.
- [ ] Every button and link does something; no decorative or placeholder controls.
- [ ] Numbers are right-aligned, tabular, with units; IDR formatted the same everywhere.
- [ ] Forecast is visibly different from actual without reading the legend.
- [ ] Empty, loading, error, and truncated states are designed, not left blank.
- [ ] Text meets contrast in both Light and Dark.
- [ ] No claim in the UI that the data cannot support (for example `customer_segment` is labeled as ground truth).

## Mockup review (2026-11-15)
Checked against the checklist above on all three screens in both themes.
- Passed: no accent-line cards (the active tab uses an underline), no gradients or emoji, one accent color, every control is functional, forecast is distinct from actual by dash, band, and origin rule, all states are designed (see States), contrast holds in Light and Dark.
- Open for the web build: tabular numerals must be enabled in CSS (the mockup font renders proportional digits); chart marks need a text or pattern cue besides color in the final charts (direct labels are drawn, dash patterns separate actual from forecast); the SQL console right rail for History is shown as a tab in the schema panel instead.

# Design principles

Working rules for the UI. A Figma mockup pass comes before the P4 screens; until then this file is the source of truth, and the tokens below are placeholders to be confirmed in that pass.

## Principles
1. The data is the interface. Charts and tables get the space; chrome stays quiet.
2. Honest by default: every number shows its unit, its filter state, and the time it covers. Forecasts never look like actuals.
3. Nothing decorative that does nothing. A button exists only if it works. No placeholder panels.
4. Dense but readable: tables right-align numbers, use tabular figures, and format IDR consistently.

## Do not
- Cards with rounded corners and a colored accent line on the left or a corner.
- Gradient hero blocks, glassmorphism, stock illustrations, emoji as icons.
- More than one accent color competing on a page.

## Tokens (placeholders)
| Token | Light | Dark | Use |
|---|---|---|---|
| `--bg` | `#fafaf8` | `#14161a` | page |
| `--fg` | `#1b1d1f` | `#e8e9e6` | text |
| `--muted` | `#5d6368` | `#9aa0a6` | secondary text |
| `--line` | `#d9dbd6` | `#2c3036` | dividers |
| Type | system sans, tabular numerals for data | | |

Categorical chart colors: pick a colorblind-safe set in the Figma pass and validate contrast in both themes; never rely on color alone (use labels or patterns for series).

## Visual language for predictions
| Element | Style |
|---|---|
| Actual | solid line |
| Forecast | dashed line, same hue as the actual series |
| Interval | translucent band around the forecast, level shown in the legend (for example 80%) |
| Forecast origin | thin vertical rule labeled with the date |
| Demo data | persistent banner while any `is_demo` run is shown |
| Model info | model name, version, `trained_until`, and `forecast_origin` visible near the chart, not in a tooltip |

## Layout
- Left navigation with the page list, a global filter bar on top (date range, season, route, channel, seat class), content below.
- Filters live in the URL so a view can be shared and reopened.
- The SQL console is a split view: editor on top, results below, schema browser on the side. Results are virtualized and show duration, row count, and a truncated notice.
- Works from 1280 px wide down to a tablet; phone is out of scope for the MVP.
